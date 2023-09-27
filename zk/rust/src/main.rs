use anyhow::Result;
use std::sync::Arc;
use std::io;
use lurk::{
    eval::{
        lang::{Coproc, Lang},
    },
    public_parameters::public_params,
    proof::nova::{NovaProver},
    proof::Prover,
    store::Store,
};
use fcomm::{ReductionCount, Proof, CommittedExpression, LurkPtr, Commitment};
use pasta_curves::pallas;
use tempfile::Builder;
use camino::Utf8Path;
use hex::FromHex;
use regex::Regex;

pub type S1 = pallas::Scalar;

use flate2::{write::ZlibEncoder, Compression};

fn main() {
    /*let result = Proof {
        claim: proof.claim.clone(),
        proof: compressed,
        reduction_count: rc,
        num_steps: proof.num_steps,
    }.verify(&pp, &lang).unwrap();

    println!("{:?}", result.verified);
    println!("{:?}", evaluation.expr);
    println!("{:?}", evaluation.expr_out);*/

    let proof = prove("(lambda (pub priv) (+ pub priv))", "3", "5").expect("proof creation failed");
    println!("Proof Len: {}", proof.len());
}

fn prove(lurk_program: &str, public_vars: &str, private_vars: &str) -> Result<Vec<u8>> {
    let program = replace_sha256_expression(lurk_program);
    let fcomm_path_key = "FCOMM_DATA_PATH";
    let tmp_dir = Builder::new().prefix("tmp").tempdir()?;
    let tmp_dir_path = Utf8Path::from_path(tmp_dir.path()).ok_or(io::Error::new(
        io::ErrorKind::InvalidData,
        "Failed to convert path to Utf8Path",
    ))?;
    let fcomm_path_val = tmp_dir_path.join("fcomm_data");
    std::env::set_var(fcomm_path_key, fcomm_path_val.clone());

    let limit = 1000;
    let lang:Lang<S1, Coproc<S1>> = Lang::new();
    let lang_rc = Arc::new(lang.clone());
    let rc = ReductionCount::Ten;
    let pp = public_params(
        rc.count(),
        true,
        lang_rc.clone(),
        &fcomm_path_val.join("public_params"),
    )
        .expect("public params");
    let s = &mut Store::<S1>::default();

    let committed_expr = CommittedExpression::<S1> {
        expr: LurkPtr::Source(private_vars.into()),
        secret: None,
        commitment: None,
    };

    let comm_ptr = committed_expr.expr_ptr(s, limit, &lang)?;

    let (commitment, _secret) = Commitment::from_ptr_with_hiding(s, &comm_ptr)?;

    let commitment_str = commitment.to_string();

    let commitment_bytes = Vec::from_hex(commitment_str.clone())?;

    let expression = format!(r#"(letrec ((do {program}))(do {public_vars} (open 0x{commitment_str})))"#);

    let prover = NovaProver::<S1, Coproc<S1>>::new(rc.count(), lang.clone());

    let input = s.read(&expression)?;

    let proof = Proof::eval_and_prove(s, input, None, limit, false, &prover, &pp, lang_rc.clone())?;
    let compressed = proof.proof.compress(&pp)?;

    let mut encoder = ZlibEncoder::new(Vec::new(), Compression::default());
    bincode::serialize_into(&mut encoder, &compressed)?;

    let compressed_snark_encoded = encoder.finish()?;
    Ok(prepend_commitment(commitment_bytes, prepend_u32_to_bytes(proof.num_steps, compressed_snark_encoded)))
}

fn prepend_u32_to_bytes(size: usize, bytes: Vec<u8>) -> Vec<u8> {
    // Convert the usize to u32. This is a safe cast if we're certain
    // the usize will never be larger than the maximum value of u32.
    // Otherwise, you might want to handle potential overflow.
    let size_as_u32 = size as u32;

    // Convert the u32 to its byte representation.
    let size_bytes = size_as_u32.to_ne_bytes();

    // Create a new Vec<u8> with enough capacity.
    let mut combined = Vec::with_capacity(size_bytes.len() + bytes.len());

    // Append the bytes of the u32.
    combined.extend_from_slice(&size_bytes);

    // Append the original byte array.
    combined.extend_from_slice(&bytes);

    combined
}

fn prepend_commitment(prefix: Vec<u8>, data: Vec<u8>) -> Vec<u8> {
    let mut combined = Vec::with_capacity(prefix.len() + data.len());

    combined.extend(prefix);
    combined.extend(data);

    combined
}

fn replace_sha256_expression(input: &str) -> String {
    let re = Regex::new(r"\(sha256 ([^\)]+)\)").unwrap();
    re.replace_all(input, "(eval (cons 'sha256 (cons $1 nil)))").to_string()
}