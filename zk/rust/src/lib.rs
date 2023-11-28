// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

use std::ffi::CStr;
use std::os::raw::{c_char, c_uchar};
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
use pasta_curves::pallas;

pub type S1 = pallas::Scalar;
const OUT_LEN: usize = 32;

/*
fn main() {
    let comm = lurk_commit("1");
    println!("{:?}", comm);
}*/

/*
pub fn lurk_commit(expr: &str, ) -> Vec<u8> {
    let store = &mut Store::<S1>::default();
    let state = State::init_lurk_state().rccell();
    let ptr = store.read(state, expr).expect("could not read expression");
    let comm = store.commit(ptr);
    comm.get_atom().unwrap().to_bytes()
}*/

#[no_mangle]
pub extern "C" fn lurk_commit(expr: *const c_char, out: *mut c_uchar) -> i32 {
    // Convert C string to Rust string
    let c_str = unsafe { CStr::from_ptr(expr) };
    let expr_str = match c_str.to_str() {
        Ok(str) => str,
        Err(_) => return -1, // Indicate error
    };

    // Your original logic here...
    let store = &mut Store::<S1>::default();
    let state = State::init_lurk_state().rccell();

    let ptr = match store.read(state, expr_str) {
        Ok(ptr) => ptr,
        Err(_) => return -1, // Indicate error
    };

    let (output, ..) = match evaluate_simple::<S1, Coproc<S1>>(None, ptr, store, 10000) {
        Ok((out, ..)) => (out, ..),
        Err(_) => return -1, // Indicate error
    };

    if output.len() < 1 {
        return -1;
    }

    let comm = store.commit(output[0]);
    let atom_bytes = match comm.get_atom() {
        Some(atom) => atom.to_bytes(),
        None => return -1, // Indicate error
    };

    // Ensure the output size matches the expected length
    if atom_bytes.len() != OUT_LEN {
        return -1; // Indicate error if length mismatch
    }

    // Copy the data into the output buffer
    unsafe {
        std::ptr::copy_nonoverlapping(atom_bytes.as_ptr(), out, OUT_LEN);
    }

    0 // Indicate success
}
