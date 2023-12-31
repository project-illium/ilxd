// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

use std::ffi::CStr;
use std::os::raw::{c_char, c_uchar};
use std::error::Error;
use lurk::{
    field::LurkField,
    lem::{
        eval::evaluate_simple,
        store::Store,
    },
    eval::{
        lang::Coproc
    },
    state::State,
};
use rand::{rngs::OsRng};
use pasta_curves::{
    pallas::Scalar as Fr,
    group::ff::Field
};

const OUT_LEN: usize = 32;

#[no_mangle]
pub extern "C" fn lurk_commit(expr: *const c_char, out: *mut c_uchar) -> i32 {
    // Convert C string to Rust string
    let c_str = unsafe { CStr::from_ptr(expr) };
    let expr_str = match c_str.to_str() {
        Ok(str) => str,
        Err(_) => return -1, // Indicate error
    };

    let store = &mut Store::<Fr>::default();
    let state = State::init_lurk_state().rccell();

    let ptr = match store.read(state, expr_str) {
        Ok(ptr) => ptr,
        Err(_) => return -1, // Indicate error
    };

    let (output, ..) = match evaluate_simple::<Fr, Coproc<Fr>>(None, ptr, store, 10000) {
        Ok((out, ..)) => (out, ..),
        Err(_) => return -1, // Indicate error
    };

    if output.len() < 1 {
        return -1;
    }
    let comm = store.commit(output[0]);
    let comm_bytes = store.hash_ptr(&comm).value().to_bytes();

    // Ensure the output size matches the expected length
    if comm_bytes.len() != OUT_LEN {
        return -1; // Indicate error if length mismatch
    }

    // Copy the data into the output buffer
    unsafe {
        std::ptr::copy_nonoverlapping(comm_bytes.as_ptr(), out, OUT_LEN);
    }

    0 // Indicate success
}

fn create_proof(lurk_program: String, private_params: String, public_params: String) -> Result<Vec<u8>, Box<dyn Error>> {
    let store = &Store::<Fr>::default();

    let secret = Fr::random(OsRng);
    let commitment_ptr = store.hide(secret, store.read_with_default_state(private_params.as_str())?);
    let commitment_zpr = store.hash_ptr(&commitment_ptr);
    let commitment: String = commitment_zpr.value().to_bytes().iter().map(|byte| format!("{:02x}", byte)).collect();

    let expr = format!(r#"(letrec ((func {lurk_program}))
        (func (open 0x{commitment}) {public_params}))"#);

    println!("{:?}", expr);
    let proof:Vec<u8> = vec![];
    Ok(proof)
}

#[cfg(test)]
mod tests {
    use crate::create_proof;

    #[test]
    fn test_prove() {
        let proof = create_proof("(lambda (priv pub) (= (car priv) (car pub)))".to_string(), "(cons 4 5)".to_string(), "(cons 4 8)".to_string());
        println!("{:?}", proof);
    }
}