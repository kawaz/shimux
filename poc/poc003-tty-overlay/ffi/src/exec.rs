/// Execute a shell command and capture its stdout output.
/// Returns a MoonBit Bytes containing the captured stdout data.
///
/// Design rationale: Uses posix_spawn instead of fork to avoid issues with
/// the MoonBit GC runtime. This follows the same pattern as pty_spawn.
#[unsafe(no_mangle)]
pub extern "C" fn shimux_exec_capture(cmd: *const u8) -> *mut u8 {
    use crate::{bytes_to_cstring, get_errno};

    let cmd_cstr = match bytes_to_cstring(cmd) {
        Some(c) => c,
        None => return unsafe { crate::moonbit_make_bytes_raw(0) },
    };

    // Create a pipe for capturing stdout
    let mut pipe_fds: [libc::c_int; 2] = [-1, -1];
    if unsafe { libc::pipe(pipe_fds.as_mut_ptr()) } != 0 {
        return unsafe { crate::moonbit_make_bytes_raw(0) };
    }
    let pipe_read = pipe_fds[0];
    let pipe_write = pipe_fds[1];

    // Set up posix_spawn file actions
    let mut file_actions: libc::posix_spawn_file_actions_t = std::ptr::null_mut();
    if unsafe { libc::posix_spawn_file_actions_init(&mut file_actions) } != 0 {
        unsafe {
            libc::close(pipe_read);
            libc::close(pipe_write);
        }
        return unsafe { crate::moonbit_make_bytes_raw(0) };
    }

    // Close pipe_read in child
    if unsafe { libc::posix_spawn_file_actions_addclose(&mut file_actions, pipe_read) } != 0 {
        unsafe {
            libc::posix_spawn_file_actions_destroy(&mut file_actions);
            libc::close(pipe_read);
            libc::close(pipe_write);
        }
        return unsafe { crate::moonbit_make_bytes_raw(0) };
    }

    // Redirect child stdout to pipe_write
    if unsafe { libc::posix_spawn_file_actions_adddup2(&mut file_actions, pipe_write, 1) } != 0 {
        unsafe {
            libc::posix_spawn_file_actions_destroy(&mut file_actions);
            libc::close(pipe_read);
            libc::close(pipe_write);
        }
        return unsafe { crate::moonbit_make_bytes_raw(0) };
    }

    // Close pipe_write after dup (it's now stdout)
    if pipe_write > 2 {
        if unsafe { libc::posix_spawn_file_actions_addclose(&mut file_actions, pipe_write) } != 0 {
            unsafe {
                libc::posix_spawn_file_actions_destroy(&mut file_actions);
                libc::close(pipe_read);
                libc::close(pipe_write);
            }
            return unsafe { crate::moonbit_make_bytes_raw(0) };
        }
    }

    // Build argv: ["/bin/sh", "-c", cmd, NULL]
    let sh_path = b"/bin/sh\0";
    let dash_c = b"-c\0";
    let argv: [*mut libc::c_char; 4] = [
        sh_path.as_ptr() as *mut libc::c_char,
        dash_c.as_ptr() as *mut libc::c_char,
        cmd_cstr.as_ptr() as *mut libc::c_char,
        std::ptr::null_mut(),
    ];

    // Get current environment
    unsafe extern "C" {
        static environ: *const *const libc::c_char;
    }

    let mut child_pid: libc::pid_t = 0;
    let spawn_ret = unsafe {
        libc::posix_spawn(
            &mut child_pid,
            sh_path.as_ptr() as *const libc::c_char,
            &file_actions,
            std::ptr::null(),
            argv.as_ptr(),
            environ as *const *mut libc::c_char,
        )
    };

    unsafe { libc::posix_spawn_file_actions_destroy(&mut file_actions) };

    if spawn_ret != 0 {
        unsafe {
            libc::close(pipe_read);
            libc::close(pipe_write);
        }
        return unsafe { crate::moonbit_make_bytes_raw(0) };
    }

    // Parent closes write end of pipe
    unsafe { libc::close(pipe_write) };

    // Read all data from pipe_read
    let mut buf = Vec::new();
    let mut tmp = [0u8; 4096];
    loop {
        let n = unsafe { libc::read(pipe_read, tmp.as_mut_ptr() as *mut libc::c_void, tmp.len()) };
        if n > 0 {
            buf.extend_from_slice(&tmp[..n as usize]);
        } else if n == 0 {
            break; // EOF
        } else {
            // Error: check if EINTR
            if get_errno() == libc::EINTR {
                continue;
            }
            break;
        }
    }

    unsafe { libc::close(pipe_read) };

    // Wait for child process to finish
    let mut status: libc::c_int = 0;
    loop {
        let ret = unsafe { libc::waitpid(child_pid, &mut status, 0) };
        if ret == -1 {
            if get_errno() == libc::EINTR {
                continue;
            }
            break;
        }
        break;
    }

    // Create MoonBit Bytes from captured data
    let len = buf.len();
    let ptr = unsafe { crate::moonbit_make_bytes_raw(len as i32) };
    if !ptr.is_null() && len > 0 {
        unsafe { std::ptr::copy_nonoverlapping(buf.as_ptr(), ptr, len) };
    }
    ptr
}
