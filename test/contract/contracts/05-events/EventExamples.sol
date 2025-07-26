// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract EventExamples {
    // Simple event
    event ValueSet(uint256 newValue);
    
    // Event with indexed parameters
    event Transfer(address indexed from, address indexed to, uint256 amount);
    
    // Event with multiple parameters
    event UserAction(address indexed user, string action, uint256 timestamp, bool success);
    
    // Event with struct parameter
    struct UserData {
        string name;
        uint256 age;
        address wallet;
    }
    event UserRegistered(address indexed user, UserData data);
    
    // Event with array parameter
    event BatchTransfer(address indexed from, address[] to, uint256[] amounts);
    
    // Anonymous event
    event AnonymousEvent() anonymous;
    
    uint256 public value;
    mapping(address => UserData) public users;
    
    function setValue(uint256 _value) public {
        value = _value;
        emit ValueSet(_value);
    }
    
    function transfer(address _to, uint256 _amount) public {
        emit Transfer(msg.sender, _to, _amount);
    }
    
    function performAction(string memory _action) public {
        bool success = true;
        emit UserAction(msg.sender, _action, block.timestamp, success);
    }
    
    function registerUser(string memory _name, uint256 _age) public {
        UserData memory userData = UserData({
            name: _name,
            age: _age,
            wallet: msg.sender
        });
        
        users[msg.sender] = userData;
        emit UserRegistered(msg.sender, userData);
    }
    
    function batchTransfer(address[] memory _recipients, uint256[] memory _amounts) public {
        require(_recipients.length == _amounts.length, "Arrays length mismatch");
        emit BatchTransfer(msg.sender, _recipients, _amounts);
    }
    
    function emitAnonymousEvent() public {
        emit AnonymousEvent();
    }
    
    // Function that emits multiple events
    function complexOperation(uint256 _value, string memory _action) public {
        emit ValueSet(_value);
        emit UserAction(msg.sender, _action, block.timestamp, true);
        emit AnonymousEvent();
    }
    
    // Function that demonstrates event with dynamic data
    function emitDynamicEvent(bytes memory _data) public {
        emit UserAction(msg.sender, string(_data), block.timestamp, true);
    }
} 