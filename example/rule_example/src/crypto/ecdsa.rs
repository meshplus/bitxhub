#![allow(unused_assignments)]
use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_void};

extern "C" {
  fn ecdsa_verify(sig_ptr: i64, digest_ptr: i64, pubkey_ptr: i64, opt: i32) -> i32;
}

pub enum EcdsaAlgorithmn {
  P256 = 1,
  Secp256k1 = 2,
}

pub fn verify(signature: &[u8], digest: &[u8], pubkey: &[u8], opt: EcdsaAlgorithmn) -> bool {
  let mut ecdsa_opt = 0;
  match opt {
    EcdsaAlgorithmn::P256 => ecdsa_opt = 1,
    EcdsaAlgorithmn::Secp256k1 => ecdsa_opt = 2,
  }
  let res = unsafe {
    ecdsa_verify(
      signature.as_ptr() as i64,
      digest.as_ptr() as i64,
      pubkey.as_ptr() as i64,
      ecdsa_opt,
    )
  };
  if res == 1 {
    return true;
  } else {
    return false;
  }
  // return true;
}
