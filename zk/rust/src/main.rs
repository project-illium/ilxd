use anyhow::Result;
use std::sync::Arc;
use std::time::Instant;
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
    let unlocking_script = "(lambda (script-params unlocking-params input-index private-params public-params) (eq (car script-params) (car unlocking-params)))";

    let private_params = "'( ;;inputs
 ((0x{script-commitment} ;;commitment
   1000000000            ;; amount
   1                     ;; asset ID
   '(234)                ;; script params
   0                     ;; commitment index
   0                     ;; state
   21                    ;; salt
   '(123)                ;; unlocking params
   ((1 t) (2 nil) (3 t) (4 t) (5 nil) (6 nil) (7 t) (8 t)) ;; accumulator hashes
   (0 1 2 3 4 5 6 7 8)))                                   ;; accumulator
 ( ;; outputs
   (999     ;; script hash
   100000  ;; amount
   1       ;; asset-iD
   0       ;; state
   5)      ;; salt
  (111
   2000000
   1
   0
   19)))";

    let public_params = "'(
                    '(72)            ;; nullifiers
                    6                ;; txo root
                    0                ;; fee
                    0                ;; coinbase
                    nil              ;; mint id
                    0                ;; mint amount
                    (                ;; public outputs
                        ( 99         ;; commitment
                        '(9 2))      ;; ciphertext
                        (44 '(11 10)))
                    5                ;; sig hash
                    0                ;; locktime
                    )";
    let proof = prove(public_params, private_params, unlocking_script).expect("proof creation failed");
    println!("Proof Len: {}", proof.len());
}

fn prove(public_vars: &str, private_vars: &str, unlocking_script: &str) -> Result<Vec<u8>> {
    //let program = replace_sha256_expression(lurk_program);
    let fcomm_path_key = "FCOMM_DATA_PATH";
    let tmp_dir = Builder::new().prefix("tmp").tempdir()?;
    let tmp_dir_path = Utf8Path::from_path(tmp_dir.path()).ok_or(io::Error::new(
        io::ErrorKind::InvalidData,
        "Failed to convert path to Utf8Path",
    ))?;
    let fcomm_path_val = tmp_dir_path.join("fcomm_data");
    std::env::set_var(fcomm_path_key, fcomm_path_val.clone());

    let limit = 100000;
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

    let committed_func = CommittedExpression::<S1> {
        expr: LurkPtr::Source(unlocking_script.into()),
        secret: None,
        commitment: None,
    };

    let func_ptr = committed_func.expr_ptr(s, limit, &lang)?;
    let (func_commitment, _secret) = Commitment::from_ptr_with_hiding(s, &func_ptr)?;
    let func_commitment_str = func_commitment.to_string();
    let private_params = private_vars.replace("{script-commitment}", &func_commitment_str);

    let committed_expr = CommittedExpression::<S1> {
        expr: LurkPtr::Source(private_params.clone().into()),
        secret: None,
        commitment: None,
    };

    let comm_ptr = committed_expr.expr_ptr(s, limit, &lang)?;
    let (commitment, _secret) = Commitment::from_ptr_with_hiding(s, &comm_ptr)?;
    let commitment_str = commitment.to_string();
    let commitment_bytes = Vec::from_hex(commitment_str.clone())?;

    //let expression = format!(lurk_program);
    let exec_str = format!(r#"(validate-transaction (open 0x{commitment_str}) {public_vars}))"#);
    let expression = lurk_program.to_owned() + &exec_str;

    let prover = NovaProver::<S1, Coproc<S1>>::new(rc.count(), lang.clone());

    let input = s.read(&expression)?;

    let proof_start = Instant::now();
    let proof = Proof::eval_and_prove(s, input, None, limit, false, &prover, &pp, lang_rc.clone())?;

    let compressed = proof.proof.compress(&pp)?;
    let proof_end = proof_start.elapsed();
    println!("Proofs took {:?}", proof_end);

    println!("{:?}", proof.claim.evaluation().unwrap().expr_out);

    let mut encoder = ZlibEncoder::new(Vec::new(), Compression::default());
    bincode::serialize_into(&mut encoder, &compressed)?;

    let compressed_snark_encoded = encoder.finish()?;
    println!("Num iterations: {:?}", proof.num_steps * rc.count());
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

static lurk_program: &str = "(
    letrec (
        ;; hash uses the blake2s function inside the circuit.
        ;; x is expected to be an integer of up to 255 bits.
        ;; It gets converted to its byte representation inside
        ;; the coprocessor before hashing.
        ;;
        ;; This is an expensive compution and hashing will
        ;; substantially increase proving time.
        (hash (lambda (x) (num (commit x))))
            ;;(eval (cons 'blake2s_hash (cons x nil))))))

        ;; cat-and-hash accepts a list of integers, each up to
        ;; 255 bits. The integers are converted to their byte
        ;; representation, concatenated together, then hashed.
        ;; If a list element is type u64 then only a 8 byte
        ;; array will be concatenated for that element.
        (cat-and-hash (lambda (list)
            (hash list)))

        ;; map-get returns an item from a map given the key.
        ;; the map must be a list of form (:key item :key item).
        (map-get (lambda (key plist)
            (if plist
                (if (eq key (car plist))
                    (car (cdr plist))
                    (map-get key (cdr (cdr plist))))
                nil)))

        ;; list-get returns the item at the given index from
        ;; the list or nil if the index doesn't exist.
        (list-get (lambda (idx plist)
            (if (= idx 0)
                (car plist)
                (list-get (- idx 1) (cdr plist)))))

        ;; map-update updates value of the given key in the
        ;; provided map and returns the new map.
        ;; If the key is not in the map a new map entry will
        ;; be added.
        ;; The map is formatted as a flat list of format
        ;; (:key item :key item)
        (map-update (lambda (key value map)
              (if (eq map nil)
                  (cons key (cons value nil))
                  (let ((existing-key (car map))
                        (rest-map (cdr map)))
                        (if (= existing-key key)
                            (cons key (cons value (cdr (cdr map))))
                            (cons existing-key (map-update key value rest-map)))))))


        ;; validate-inclusion-proof validates that the provided
        ;; output commitment connects to the provided merkle root
        ;; via a merkle inclusion proof.
        (validate-inclusion-proof (lambda (output-commitment commitment-index hashes accumulator root)
            (letrec (
                (h (cat-and-hash '(output-commitment (u64 commitment-index))))
                (is-hash-in-accumulator (lambda (h accumulator)
                    (if (= h (car accumulator))
                        t
                        (if (eq (cdr accumulator) nil)
                            nil
                            (is-hash-in-accumulator h (cdr accumulator))))))
                (hash-branches (lambda (h hashes)
                    (let ((next-hash (car hashes))
                        (val (car next-hash))
                        (left (cdr next-hash))
                        (new-h (if (eq left t)
                                (cat-and-hash '(val h))
                                (cat-and-hash '(h val)))))

                        (if (eq (cdr hashes) nil)
                            new-h
                            (hash-branches new-h (cdr hashes)))))))

                (if (is-hash-in-accumulator (hash-branches 'h hashes) accumulator)
                    (eq (cat-and-hash accumulator) root)
                    nil))))

        ;; validate-input validates the input against the provided
        ;; parameters. In particular it validates that the inclusion
        ;; proof is valid, the nullifier is calculated correctly,
        ;; and the committed unlocking function executed correctly.
        (validate-input (lambda (input idx txo-root nullifiers private-params public-params)
            (let (
                (script-commitment (car input))
                (amount (cdr input))
                (asset-id (cdr amount))
                (script-params (cdr asset-id))
                (commitment-idx (cdr script-params))
                (state (cdr commitment-idx))
                (salt (cdr state))
                (unlocking-params (cdr salt))
                (inclusion-proof-hashes (cdr unlocking-params))
                (inclusion-proof-acc (cdr inclusion-proof-hashes))
                (spend-script-preimage (cons script-commitment (car script-params)))
                (script-hash (hash 'spend-script-preimage))
                (nullifier (list-get idx nullifiers))
                (calculated-nullifier (cat-and-hash '((car commitment-idx) (car salt) script-commitment (car script-params))))
                (output-commitment (cat-and-hash '(script-hash (car amount) (car asset-id) (car state) (car salt)))))

                (if (eq ((open (comm script-commitment))
                        (car script-params)
                        (car unlocking-params)
                        idx
                        private-params
                        public-params) nil)
                    nil
                    (if (eq (validate-inclusion-proof output-commitment
                                (car commitment-idx)
                                (car inclusion-proof-hashes)
                                (car inclusion-proof-acc)
                                txo-root) nil)
                        nil
                        (= nullifier calculated-nullifier))))))

        ;; validate-output validates that the provided private output
        ;; data matches the public output commitment found in the
        ;; transaction.
        (validate-output (lambda (output idx public-outputs)
                    (let (
                        (script-hash (car output))
                        (amount (cdr output))
                        (asset-id (cdr amount))
                        (state (cdr asset-id))
                        (salt (cdr state))
                        (public-output (list-get idx public-outputs))
                        (commitment-preimage (cat-and-hash '(
                                                    script-hash
                                                    (u64 (car amount))
                                                    (car asset-id)
                                                    (car state)
                                                    (car salt))))
                        (output-commitment (hash commitment-preimage)))
                        (= output-commitment (car public-output)))))

        ;; validate-amounts validates that the total private input
        ;; values do not exceed the total private output values.
        ;; It performs this check both for the illium asset ID and
        ;; for other token asset IDs. It will return false if an
        ;; integer overflow is detected.
        (validate-amounts (lambda (private-inputs private-outputs coinbase fee mint-id mint-amount)
                (letrec (
                    (check-overflow (lambda (a b)
                            (if (> b 0)
                                (if (> a (- 18446744073709551615 b))
                                    t
                                    nil)
                                nil)))
                    (validate-assets (lambda (in-map out-map)
                               (let (
                                     (out-asset-id (car out-map))
                                     (in-value (map-get out-asset-id in-map)))

                                      (if (eq out-asset-id nil)
                                          t
                                          (if (eq in-value nil)
                                              (if (eq out-asset-id mint-id)
                                                  (<= (car (cdr out-map)) mint-amount)
                                                  nil)
                                              (if (> (car (cdr out-map)) (+ in-value mint-amount))
                                                  nil
                                                  (validate-assets in-map (cdr out-map))))))))

                    (sum-xputs (lambda (xputs illium-sum asset-map)
                                    (letrec (
                                        (amount (cdr (car xputs)))
                                        (asset-id (car (cdr amount)))
                                        (sum-assets (lambda (asset-map a-id amt)
                                                        (let ((asset-amt (map-get a-id asset-map)))

                                                        (if (eq asset-amt nil)
                                                            (map-update a-id amt asset-map)
                                                            (if (check-overflow asset-amt amt)
                                                                nil
                                                                (map-update a-id (+ asset-amt amt) asset-map))) ))))

                                        (if (eq xputs nil)
                                            (cons illium-sum asset-map)
                                            (if (= asset-id 0)
                                                (if (check-overflow illium-sum (car amount))
                                                      nil
                                                      (sum-xputs (cdr xputs) (+ illium-sum (car amount)) asset-map))
                                                (let ((new-asset-map (sum-assets asset-map asset-id (car amount))))
                                                   (if (eq new-asset-map nil)
                                                        nil
                                                        (sum-xputs (cdr xputs) illium-sum new-asset-map) )))))))

                    (in-vals (sum-xputs private-inputs 0 nil))
                    (out-vals (sum-xputs private-outputs 0 nil)))

                    (if (eq in-vals nil)
                        nil
                        (if (eq out-vals nil)
                            nil
                            (if (check-overflow (car in-vals) coinbase)
                                nil
                                (if (check-overflow (car out-vals) fee)
                                    nil
                                    (if (> (+ (car out-vals) fee) (+ (car in-vals) coinbase))
                                        nil
                                        (validate-assets (cdr in-vals) (cdr out-vals))))))))))

        (validate-transaction (lambda (private-params public-params)
            (letrec (
                    (private-inputs (car private-params))
                    (private-outputs (car (cdr private-params)))
                    (nullifiers (car public-params))
                    (txo-root (cdr public-params))
                    (fee (cdr txo-root))
                    (coinbase (cdr fee))
                    (mint-id (cdr coinbase))
                    (mint-amount (cdr mint-id))
                    (public-outputs (cdr mint-amount))

                    (validate-inputs (lambda (idx inputs)
                        (if (eq inputs nil)
                            t
                            (if (validate-input (car inputs) idx (car txo-root) nullifiers private-params public-params)
                                (validate-inputs (+ idx 1) (cdr inputs))
                                nil))))
                    (validate-outputs (lambda (idx outputs)
                            (if (eq outputs nil)
                                t
                                (if (validate-output (car outputs) idx (car public-outputs))
                                    (validate-outputs (+ idx 1) (cdr outputs))
                                    nil)))))

                   (validate-amounts private-inputs private-outputs (car coinbase) (car fee) (car mint-id) (car mint-amount)))))
                   ;;(validate-inputs 0 private-inputs))))
                   ;;(if (eq private-inputs nil)
                   ;;    nil
                   ;;    (if (validate-inputs 0 private-inputs)
                   ;;        (if (eq private-outputs nil)
                   ;;            nil
                   ;;            (if (validate-outputs 0 private-outputs)
                   ;;               (validate-amounts private-inputs private-outputs (car coinbase) (car fee) (car mint-id) (car mint-amount))
                   ;;                nil))
                   ;;        nil)))))
    )";