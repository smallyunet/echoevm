// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract StructTypes {
    // Basic struct
    struct Person {
        string name;
        uint256 age;
        address wallet;
    }
    
    // Struct with nested struct
    struct Address {
        string street;
        string city;
        string country;
    }
    
    struct Employee {
        Person person;
        Address addr;
        uint256 salary;
        bool isActive;
    }
    
    // Struct with array
    struct Team {
        string name;
        Person[] members;
        uint256 totalMembers;
    }
    
    // Struct with mapping
    struct Company {
        string name;
        mapping(address => Employee) employees;
        uint256 employeeCount;
    }
    
    // State variables
    Person public owner;
    Employee public manager;
    Team public developmentTeam;
    
    constructor() {
        owner = Person("John Doe", 30, msg.sender);
        
        Address memory managerAddr = Address("123 Main St", "New York", "USA");
        manager = Employee(owner, managerAddr, 5000, true);
        
        developmentTeam.name = "Dev Team";
        developmentTeam.totalMembers = 0;
    }
    
    function createPerson(string memory _name, uint256 _age, address _wallet) public pure returns (Person memory) {
        return Person(_name, _age, _wallet);
    }
    
    function updatePersonName(string memory _name) public {
        owner.name = _name;
    }
    
    function updatePersonAge(uint256 _age) public {
        owner.age = _age;
    }
    
    function getPersonInfo() public view returns (string memory, uint256, address) {
        return (owner.name, owner.age, owner.wallet);
    }
    
    function createEmployee(
        string memory _name,
        uint256 _age,
        address _wallet,
        string memory _street,
        string memory _city,
        string memory _country,
        uint256 _salary
    ) public pure returns (Employee memory) {
        Person memory person = Person(_name, _age, _wallet);
        Address memory addr = Address(_street, _city, _country);
        return Employee(person, addr, _salary, true);
    }
    
    function updateEmployeeSalary(uint256 _salary) public {
        manager.salary = _salary;
    }
    
    function getEmployeeInfo() public view returns (
        string memory,
        uint256,
        address,
        string memory,
        string memory,
        string memory,
        uint256,
        bool
    ) {
        return (
            manager.person.name,
            manager.person.age,
            manager.person.wallet,
            manager.addr.street,
            manager.addr.city,
            manager.addr.country,
            manager.salary,
            manager.isActive
        );
    }
    
    function addTeamMember(string memory _name, uint256 _age, address _wallet) public {
        Person memory newMember = Person(_name, _age, _wallet);
        developmentTeam.members.push(newMember);
        developmentTeam.totalMembers++;
    }
    
    function getTeamMember(uint256 index) public view returns (string memory, uint256, address) {
        require(index < developmentTeam.members.length, "Index out of bounds");
        Person memory member = developmentTeam.members[index];
        return (member.name, member.age, member.wallet);
    }
    
    function getTeamInfo() public view returns (string memory, uint256) {
        return (developmentTeam.name, developmentTeam.totalMembers);
    }
    
    function removeTeamMember(uint256 index) public {
        require(index < developmentTeam.members.length, "Index out of bounds");
        
        // Move last element to the position to be deleted
        developmentTeam.members[index] = developmentTeam.members[developmentTeam.members.length - 1];
        developmentTeam.members.pop();
        developmentTeam.totalMembers--;
    }
    
    function updateTeamName(string memory _name) public {
        developmentTeam.name = _name;
    }
    
    function getTeamMemberCount() public view returns (uint256) {
        return developmentTeam.members.length;
    }
    
    function isTeamEmpty() public view returns (bool) {
        return developmentTeam.members.length == 0;
    }
} 