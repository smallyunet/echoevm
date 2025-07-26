// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Switch {
    enum DayOfWeek { Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday }
    
    DayOfWeek public currentDay = DayOfWeek.Monday;
    
    function setDay(DayOfWeek _day) public {
        currentDay = _day;
    }
    
    function getDayName() public view returns (string memory) {
        if (currentDay == DayOfWeek.Monday) {
            return "Monday";
        } else if (currentDay == DayOfWeek.Tuesday) {
            return "Tuesday";
        } else if (currentDay == DayOfWeek.Wednesday) {
            return "Wednesday";
        } else if (currentDay == DayOfWeek.Thursday) {
            return "Thursday";
        } else if (currentDay == DayOfWeek.Friday) {
            return "Friday";
        } else if (currentDay == DayOfWeek.Saturday) {
            return "Saturday";
        } else if (currentDay == DayOfWeek.Sunday) {
            return "Sunday";
        } else {
            return "Unknown";
        }
    }
    
    function isWeekend() public view returns (bool) {
        if (currentDay == DayOfWeek.Saturday || currentDay == DayOfWeek.Sunday) {
            return true;
        } else {
            return false;
        }
    }
    
    function getDayNumber() public view returns (uint256) {
        if (currentDay == DayOfWeek.Monday) {
            return 1;
        } else if (currentDay == DayOfWeek.Tuesday) {
            return 2;
        } else if (currentDay == DayOfWeek.Wednesday) {
            return 3;
        } else if (currentDay == DayOfWeek.Thursday) {
            return 4;
        } else if (currentDay == DayOfWeek.Friday) {
            return 5;
        } else if (currentDay == DayOfWeek.Saturday) {
            return 6;
        } else if (currentDay == DayOfWeek.Sunday) {
            return 7;
        } else {
            return 0;
        }
    }
    
    function getNextDay() public view returns (DayOfWeek) {
        if (currentDay == DayOfWeek.Monday) {
            return DayOfWeek.Tuesday;
        } else if (currentDay == DayOfWeek.Tuesday) {
            return DayOfWeek.Wednesday;
        } else if (currentDay == DayOfWeek.Wednesday) {
            return DayOfWeek.Thursday;
        } else if (currentDay == DayOfWeek.Thursday) {
            return DayOfWeek.Friday;
        } else if (currentDay == DayOfWeek.Friday) {
            return DayOfWeek.Saturday;
        } else if (currentDay == DayOfWeek.Saturday) {
            return DayOfWeek.Sunday;
        } else if (currentDay == DayOfWeek.Sunday) {
            return DayOfWeek.Monday;
        } else {
            return DayOfWeek.Monday;
        }
    }
    
    function getPreviousDay() public view returns (DayOfWeek) {
        if (currentDay == DayOfWeek.Monday) {
            return DayOfWeek.Sunday;
        } else if (currentDay == DayOfWeek.Tuesday) {
            return DayOfWeek.Monday;
        } else if (currentDay == DayOfWeek.Wednesday) {
            return DayOfWeek.Tuesday;
        } else if (currentDay == DayOfWeek.Thursday) {
            return DayOfWeek.Wednesday;
        } else if (currentDay == DayOfWeek.Friday) {
            return DayOfWeek.Thursday;
        } else if (currentDay == DayOfWeek.Saturday) {
            return DayOfWeek.Friday;
        } else if (currentDay == DayOfWeek.Sunday) {
            return DayOfWeek.Saturday;
        } else {
            return DayOfWeek.Sunday;
        }
    }
    
    function isWorkDay() public view returns (bool) {
        if (currentDay == DayOfWeek.Monday || 
            currentDay == DayOfWeek.Tuesday || 
            currentDay == DayOfWeek.Wednesday || 
            currentDay == DayOfWeek.Thursday || 
            currentDay == DayOfWeek.Friday) {
            return true;
        } else {
            return false;
        }
    }
} 