// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract MappingTypes {
    // Simple mapping
    mapping(address => uint256) public balances;
    
    // Nested mapping
    mapping(address => mapping(address => uint256)) public allowances;
    
    // Mapping with struct
    struct UserInfo {
        string name;
        uint256 age;
        bool isActive;
    }
    mapping(address => UserInfo) public users;
    
    // Mapping with array
    mapping(address => uint256[]) public userScores;
    
    // Mapping with enum
    enum UserStatus { Inactive, Active, Suspended }
    mapping(address => UserStatus) public userStatus;
    
    constructor() {
        balances[msg.sender] = 1000;
        users[msg.sender] = UserInfo("Owner", 25, true);
        userStatus[msg.sender] = UserStatus.Active;
    }
    
    function setBalance(address _user, uint256 _amount) public {
        balances[_user] = _amount;
    }
    
    function getBalance(address _user) public view returns (uint256) {
        return balances[_user];
    }
    
    function setAllowance(address _owner, address _spender, uint256 _amount) public {
        allowances[_owner][_spender] = _amount;
    }
    
    function getAllowance(address _owner, address _spender) public view returns (uint256) {
        return allowances[_owner][_spender];
    }
    
    function setUserInfo(address _user, string memory _name, uint256 _age, bool _isActive) public {
        users[_user] = UserInfo(_name, _age, _isActive);
    }
    
    function getUserInfo(address _user) public view returns (string memory, uint256, bool) {
        UserInfo memory user = users[_user];
        return (user.name, user.age, user.isActive);
    }
    
    function addUserScore(address _user, uint256 _score) public {
        userScores[_user].push(_score);
    }
    
    function getUserScores(address _user) public view returns (uint256[] memory) {
        return userScores[_user];
    }
    
    function getUserScoreCount(address _user) public view returns (uint256) {
        return userScores[_user].length;
    }
    
    function setUserStatus(address _user, UserStatus _status) public {
        userStatus[_user] = _status;
    }
    
    function getUserStatus(address _user) public view returns (UserStatus) {
        return userStatus[_user];
    }
    
    function isUserActive(address _user) public view returns (bool) {
        return userStatus[_user] == UserStatus.Active;
    }
    
    function deleteUser(address _user) public {
        delete balances[_user];
        delete users[_user];
        delete userScores[_user];
        delete userStatus[_user];
    }
    
    function transfer(address _from, address _to, uint256 _amount) public {
        require(balances[_from] >= _amount, "Insufficient balance");
        require(allowances[_from][msg.sender] >= _amount, "Insufficient allowance");
        
        balances[_from] -= _amount;
        balances[_to] += _amount;
        allowances[_from][msg.sender] -= _amount;
    }
} 