// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

use std::{
    os::raw::{c_char, c_uchar},
    ffi::{CStr},
    error::Error,
    sync::Arc,
    ptr,
    slice,
};
use once_cell::sync::OnceCell;
use lurk::{
    eval::lang::{Lang, Coproc},
    field::LurkField,
    lem::{
        eval::{evaluate, evaluate_simple, make_eval_step_from_config, EvalConfig},
        multiframe::MultiFrame,
        store::Store,
    },
    proof::{supernova::{SuperNovaProver, PublicParams, Proof}, Prover, RecursiveSNARKTrait},
    public_parameters::{
        instance::{Instance, Kind},
        supernova_public_params,
    },
    state::{user_sym},
};
use rand::{rngs::OsRng};
use pasta_curves::{
    pallas::Scalar as Fr,
    group::ff::Field
};
use flate2::{write::ZlibEncoder, read::ZlibDecoder, Compression};
use nova::supernova::error::SuperNovaError;

use coprocessors::{
    xor::MultiCoproc,
    xor::XorCoprocessor,
    blake2s::Blake2sCoprocessor,
    checksig::ChecksigCoprocessor
};
mod coprocessors;

use lazy_static::lazy_static;

const OUT_LEN: usize = 32;
const REDUCTION_COUNT: usize = 10;

lazy_static! {
    static ref IO_ZERO: Fr = Fr::zero();
    static ref IO_ONE: Fr = Fr::one();
    static ref IO_ENV_HASH: Fr = Fr::from_bytes(&hex::decode("27345e8c5736d418e5a91fa58d8e6b220b682c6501734ad8a3841adc730de72c").unwrap()).unwrap();
    static ref IO_TWO: Fr = Fr::from_u64(2);
    static ref IO_TRUE_HASH: Fr = Fr::from_bytes(&hex::decode("5698a149855d3a8b3ac99e32b65ce146f00163130070c245a2262b46c5dbc804").unwrap()).unwrap();
    static ref IO_CONT_HASH: Fr = Fr::from_bytes(&hex::decode("1c6b873ac13018a8332a6c340d61b4834698bb84fe5680523ce546705217f40e").unwrap()).unwrap();
    static ref IO_IN_CONT_TAG: Fr = Fr::from_u64(4096);
    static ref IO_OUT_CONT_TAG: Fr = Fr::from_u64(4110);
}


#[no_mangle]
pub extern "C" fn load_public_params() {
    let _ = get_public_params();
}

#[no_mangle]
pub extern "C" fn lurk_commit(expr: *const c_char, out: *mut c_uchar) -> i32 {
    // Convert C string to Rust string
    let c_str = unsafe { CStr::from_ptr(expr) };
    let expr_str = match c_str.to_str() {
        Ok(str) => str,
        Err(_) => return -1, // Indicate error
    };

    let store = &mut Store::<Fr>::default();
    let ptr = match store.read_with_default_state(expr_str) {
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

#[no_mangle]
pub extern "C" fn create_proof_ffi(
    lurk_program: *const c_char,
    private_params: *const c_char,
    public_params: *const c_char,
    proof: *mut u8,
    proof_len: *mut usize,
    output_tag: *mut u8,
    output_val: *mut u8,
) -> i32 {
    let c_str1 = unsafe { CStr::from_ptr(lurk_program) };
    let program_str = match c_str1.to_str() {
        Ok(str) => str,
        Err(_) => return -1, // Indicate error
    };
    let c_str2 = unsafe { CStr::from_ptr(private_params) };
    let priv_params_str = match c_str2.to_str() {
        Ok(str) => str,
        Err(_) => return -1, // Indicate error
    };
    let c_str3 = unsafe { CStr::from_ptr(public_params) };
    let pub_params_str = match c_str3.to_str() {
        Ok(str) => str,
        Err(_) => return -1, // Indicate error
    };

    match create_proof(
        program_str.to_string(),
        priv_params_str.to_string(),
        pub_params_str.to_string()
    ) {
        Ok((vec1, vec2, vec3)) => {
            // Assume output1, output2, and output3 are large enough to hold the data
            unsafe {
                ptr::copy_nonoverlapping(vec1.as_ptr(), proof, vec1.len());
                *proof_len = vec1.len();
                ptr::copy_nonoverlapping(vec2.as_ptr(), output_tag, vec2.len());
                ptr::copy_nonoverlapping(vec3.as_ptr(), output_val, vec3.len());
            }
            0 // Success
        }
        Err(_) => -1, // Error
    }
}

#[no_mangle]
pub extern "C" fn verify_proof_ffi(
    lurk_program: *const c_char,
    public_params: *const c_char,
    packed_proof: *const u8,
    proof_size: usize
) -> i32 {
    let c_str1 = unsafe { CStr::from_ptr(lurk_program) };
    let program_str = match c_str1.to_str() {
        Ok(str) => str,
        Err(_) => return -1, // Indicate error
    };
    let c_str2 = unsafe { CStr::from_ptr(public_params) };
    let pub_params_str = match c_str2.to_str() {
        Ok(str) => str,
        Err(_) => return -1, // Indicate error
    };

    let proof_slice = unsafe {
        slice::from_raw_parts(packed_proof, proof_size)
    };

    let (commitment, proof) = proof_slice.split_at(32);
    let mut commitment_vec = commitment.to_vec();
    commitment_vec.reverse();

    let res = match verify_proof(
        program_str.to_string(),
        commitment_vec,
        pub_params_str.to_string(),
        proof.to_vec()) {
        Ok(res) => res,
        Err(err) => {
            return -1
        }
    };
    if res {
        return 0
    }
    1
}

static PUBLIC_PARAMS: OnceCell<Arc<PublicParams<Fr, MultiFrame<'static, Fr, MultiCoproc<Fr>>>>> = OnceCell::new();

fn get_public_params() -> Arc<PublicParams<Fr, MultiFrame<'static, Fr, MultiCoproc<Fr>>>> {
    PUBLIC_PARAMS.get_or_init(|| Arc::new(create_public_params())).clone()
}

fn create_public_params() -> PublicParams<Fr, MultiFrame<'static, Fr, MultiCoproc<Fr>>> {
    let cproc_sym_xor = user_sym(".lurk.xor");
    let cproc_sym_checksig = user_sym(".lurk.checksig");
    let cproc_sym_blake2s = user_sym(".lurk.blake2s");

    let mut lang = Lang::<Fr, MultiCoproc<Fr>>::new();
    lang.add_coprocessor(cproc_sym_xor, XorCoprocessor::new());
    lang.add_coprocessor(cproc_sym_checksig, ChecksigCoprocessor::new());
    lang.add_coprocessor(cproc_sym_blake2s, Blake2sCoprocessor::new());
    let lang_rc = Arc::new(lang.clone());

    let instance_primary = Instance::new(REDUCTION_COUNT, lang_rc, true, Kind::SuperNovaAuxParams);
    let pp = supernova_public_params::<_, _, MultiFrame<'_, _, _>>(&instance_primary).unwrap();
    pp
}

fn create_proof(lurk_program: String, private_params: String, public_params: String) -> Result<(Vec<u8>, Vec<u8>, Vec<u8>), Box<dyn Error>> {
    let store = &Store::<Fr>::default();

    let max_steps = 100000000;

    let secret = Fr::random(OsRng);
    let priv_expr = store.read_with_default_state(private_params.as_str())?;
    let (output, ..) = evaluate_simple::<Fr, MultiCoproc<Fr>>(None, priv_expr, store, max_steps)?;
    let comm = store.hide(secret, output[0]);
    let commitment_zpr = store.hash_ptr(&comm);
    let commitment_bytes = commitment_zpr.value().to_bytes();
    let commitment: String = commitment_bytes.iter().rev().map(|byte| format!("{:02x}", byte)).collect();

    let expr = format!(r#"(letrec ((f {lurk_program}))(f (open 0x{commitment}) {public_params}))"#);

    let cproc_sym_xor = user_sym(".lurk.xor");
    let cproc_sym_checksig = user_sym(".lurk.checksig");
    let cproc_sym_blake2s = user_sym(".lurk.blake2s");

    let call = store.read_with_default_state(expr.as_str())?;

    let mut lang = Lang::<Fr, MultiCoproc<Fr>>::new();
    lang.add_coprocessor(cproc_sym_xor, XorCoprocessor::new());
    lang.add_coprocessor(cproc_sym_checksig, ChecksigCoprocessor::new());
    lang.add_coprocessor(cproc_sym_blake2s, Blake2sCoprocessor::new());
    let lang_rc = Arc::new(lang.clone());

    let lurk_step = make_eval_step_from_config(&EvalConfig::new_nivc(&lang));
    let frames = evaluate(Some((&lurk_step, &lang)), call, store, max_steps).unwrap();

    let supernova_prover = SuperNovaProver::<Fr, MultiCoproc<Fr>, MultiFrame<'_, _, _>>::new(
        REDUCTION_COUNT,
        lang_rc.clone(),
    );

    let pp = get_public_params();

    let (proof, z0, zi, _num_steps) = supernova_prover.prove(&pp, &frames, store)?;
    let compressed_proof = proof.compress(&pp).unwrap();

    let mut ret_tag = zi[0].to_bytes();
    let mut ret_val = zi[1].to_bytes();
    ret_tag.reverse();
    ret_val.reverse();

    let mut encoder = ZlibEncoder::new(Vec::new(), Compression::default());
    bincode::serialize_into(&mut encoder, &compressed_proof)?;
    let compressed_snark_encoded = encoder.finish()?;

    let mut combined_proof = Vec::new();
    combined_proof.extend(commitment_bytes);
    combined_proof.extend(compressed_snark_encoded);

    Ok((combined_proof, ret_tag, ret_val))
}

fn verify_proof(lurk_program: String, commitment_bytes: Vec<u8>, public_params: String, proof: Vec<u8>) -> Result<bool, Box<dyn Error>> {
    let commitment: String = commitment_bytes.iter().map(|byte| format!("{:02x}", byte)).collect();

    let expr = format!(r#"(letrec ((f {lurk_program}))(f (open 0x{commitment}) {public_params}))"#);

    let store = &Store::<Fr>::default();
    let call = store.read_with_default_state(expr.as_str())?;
    let call_zptr = store.hash_ptr(&call);

    let mut z0: Vec<Fr> = Vec::with_capacity(6);
    z0.push(IO_ONE.clone());
    z0.push(*call_zptr.value());
    z0.push(IO_ZERO.clone());
    z0.push(IO_ENV_HASH.clone());
    z0.push(IO_IN_CONT_TAG.clone());
    z0.push(IO_CONT_HASH.clone());

    let mut zi: Vec<Fr> = Vec::with_capacity(6);
    zi.push(IO_TWO.clone());
    zi.push(IO_TRUE_HASH.clone());
    zi.push(IO_ZERO.clone());
    zi.push(IO_ENV_HASH.clone());
    zi.push(IO_OUT_CONT_TAG.clone());
    zi.push(IO_CONT_HASH.clone());

    let pp = get_public_params();
    let decoder = ZlibDecoder::new(&proof[..]);
    let decompressed_proof: Proof<Fr, MultiCoproc<Fr>, MultiFrame<Fr, MultiCoproc<Fr>>> = bincode::deserialize_from(decoder)?;
    let res = decompressed_proof.verify(&pp, &z0, &zi)?;
    Ok(res)
}

#[cfg(test)]
mod tests {
    use crate::{create_proof, verify_proof, get_public_params};

    #[test]
    fn test_prove() {
        get_public_params();
        let (packed_proof, tag, output) = create_proof(
            "(lambda (priv pub) (eq (cdr priv) (cdr pub)))".to_string(),
            "(cons 7 8)".to_string(),
            "(cons 7 8)".to_string()
        ).expect("create_proof failed");
        let mut commitment = packed_proof[..32].to_vec();
        commitment.reverse();
        let proof = &packed_proof[32..];
        let res = verify_proof(
            "(lambda (priv pub) (eq (cdr priv) (cdr pub)))".to_string(),
            commitment,
            "(cons 7 8)".to_string(),
            proof.to_vec()
        ).expect("verify_proof failed");
        println!("{:?}", res);
        assert!(res, "Verification failed");
    }
}