use std::ffi::CStr;
use std::os::raw::{c_char, c_void};

pub mod app;
pub mod crypto;
pub mod memory;
pub mod model;

use app::contract;

#[no_mangle]
pub extern "C" fn allocate(size: usize) -> *mut c_void {
  return memory::allocate(size);
}

#[no_mangle]
pub extern "C" fn deallocate(pointer: *mut c_void, capacity: usize) {
  return memory::deallocate(pointer, capacity);
}

#[no_mangle]
pub extern "C" fn start_verify(proof_ptr: *mut c_char, validator_ptr: *mut c_char) -> i32 {
  let proof = unsafe { CStr::from_ptr(proof_ptr).to_bytes() };
  let validator = unsafe { CStr::from_ptr(validator_ptr).to_bytes() };
  let res = contract::verify(proof, validator);

  return res as i32;
  // 1
}
