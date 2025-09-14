// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// Test contract to exercise LOG0-LOG4 semantics. Solidity limits events to at most
// 3 indexed parameters + the implicit first topic (event signature) unless the
// event is marked anonymous. To approximate LOG4 we use an anonymous event with
// 3 indexed params and one value placed in data (so total topics = 3 plus no
// signature topic) and a separate event to show standard signature topic usage.
contract EmitEvents {
    // Standard (non-anonymous) events: signature hash occupies topic0 automatically.
    event NoTopics();                                   // results in 1 topic (signature) -> LOG1 equivalent
    event OneTopic(uint256 indexed a);                  // signature + 1 indexed -> 2 topics (LOG2 equivalent)
    event TwoTopics(uint256 indexed a, uint256 indexed b); // signature + 2 indexed -> 3 topics (LOG3 equivalent)
    event ThreeTopics(uint256 indexed a, uint256 indexed b, uint256 indexed c); // signature + 3 indexed -> 4 topics (LOG4 equivalent)

    // Anonymous event removes the signature topic so we can control raw topic count explicitly.
    event AnonymousThree(uint256 indexed a, uint256 indexed b, uint256 indexed c) anonymous; // exactly 3 topics (LOG3)

    // NOTE: True raw LOG0 emission via high-level Solidity requires inline assembly.
    // For now we rely on at least one topic from signature for non-anonymous events.

    function fireAll() external {
        emit NoTopics();              // 1 topic (sig)
        emit OneTopic(1);             // 2 topics
        emit TwoTopics(2, 3);         // 3 topics
        emit ThreeTopics(4, 5, 6);    // 4 topics
        emit AnonymousThree(7, 8, 9); // 3 topics (anonymous)
    }
}
