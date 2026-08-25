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
