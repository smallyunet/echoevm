use super::*;

#[test]
fn discovers_selector_and_infers_storage_write_origin() {
    // Dispatcher for 0x11223344 -> pc 0x0d, then SSTORE(slot=1,value=calldata.arg0).
    let bytecode = hex::decode("600035631122334414600d57005b60043560015500").unwrap();
    let document = infer_behavior(&bytecode);
    assert_eq!(document.schema, BEHAVIOR_SCHEMA);
    assert_eq!(document.coverage.recognized_selectors, 1);
    let function = &document.functions[0];
    assert_eq!(function.selector, "0x11223344");
    assert_eq!(function.entry_pc, 13);
    let write = function
        .effects
        .iter()
        .find(|effect| effect.kind == "storage-write")
        .unwrap();
    assert_eq!(write.inputs["slot"], "constant(0x1)");
    assert_eq!(write.inputs["value"], "calldata.arg0");
    assert!(
        function
            .capabilities
            .contains(&"writes-persistent-state".into())
    );
    assert!(
        document
            .functions
            .iter()
            .find(|function| function.selector == "fallback")
            .unwrap()
            .effects
            .is_empty(),
        "selector effects should not be duplicated in the fallback summary"
    );
}

#[test]
fn infers_caller_guard_and_delegate_target() {
    // entry: CALLER == SLOAD(0), branch, then DELEGATECALL to calldata.arg0.
    let bytecode = hex::decode(
        "60003563aabbccdd14600d57005b6000543314601757005b60006000600060006004356000f400",
    )
    .unwrap();
    let document = infer_behavior(&bytecode);
    let function = &document.functions[0];
    assert!(!function.guards.is_empty());
    assert!(function.guards[0].condition.contains("caller"));
    let delegate = function
        .effects
        .iter()
        .find(|effect| effect.kind == "delegate-call")
        .unwrap();
    assert_eq!(delegate.inputs["target"], "calldata.arg0");
}

#[test]
fn reports_dynamic_jump_as_unresolved() {
    let document = infer_behavior(&hex::decode("3556").unwrap());
    assert_eq!(document.coverage.unresolved_jumps, 1);
    assert_eq!(document.functions[0].coverage.recognized_selectors, 0);
    assert!(!document.coverage.truncated);
}

#[test]
fn does_not_treat_dispatch_mask_as_a_selector() {
    let document = infer_behavior(&hex::decode("63ffffffff1680631122334414601157005b00").unwrap());
    assert_eq!(document.coverage.recognized_selectors, 1);
    assert_eq!(document.functions[0].selector, "0x11223344");
}

#[test]
fn proxy_runtime_includes_fallback_delegate_call() {
    // Runtime bytecode from 0xb07aaBc136EaB64994d3f226c88dd907dF3bf291.
    // The public IMPL() getter uses a selector, while proxy forwarding lives in fallback.
    let bytecode = hex::decode(concat!(
        "608060405260043610156049575b5f36818037808036817f000000000000000000000000",
        "9a1c1e8ebd7e50a1280a31d736388a50f3d96a4d5af43d82803e156045573d90f35b3d",
        "90fd5b5f803560e01c6356973ee514605d5750600d565b34609f5780600319360112609f",
        "577f0000000000000000000000009a1c1e8ebd7e50a1280a31d736388a50f3d96a4d60",
        "01600160a01b03166080908152602090f35b80fdfea26469706673582212205ea09bd3a4e8",
        "60c24fc3b976d42dd00080f7d4f22863c3c0627037a80672527564736f6c63430008140033"
    ))
    .unwrap();

    let document = infer_behavior(&bytecode);
    assert_eq!(document.coverage.recognized_selectors, 1);
    assert!(
        document
            .functions
            .iter()
            .any(|function| function.selector == "fallback")
    );
    let delegate = document
        .contract_effects
        .iter()
        .find(|effect| effect.kind == "delegate-call")
        .expect("fallback delegatecall");
    assert_eq!(
        delegate.inputs["target"],
        "constant(0x9a1c1e8ebd7e50a1280a31d736388a50f3d96a4d)"
    );
    assert!(
        document
            .contract_capabilities
            .contains(&"executes-delegate-code".into())
    );
}
