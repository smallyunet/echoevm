import { expect } from "chai";
import { ethers } from "hardhat";

describe("IfElse", function () {
  let ifElse: any;

  beforeEach(async function () {
    const IfElseFactory = await ethers.getContractFactory("IfElse");
    ifElse = await IfElseFactory.deploy();
    await ifElse.waitForDeployment();
  });

  describe("Simple If", function () {
    it("should return correct message for values greater than 100", async function () {
      expect(await ifElse.simpleIf(150)).to.equal("Greater than 100");
      expect(await ifElse.simpleIf(200)).to.equal("Greater than 100");
    });

    it("should return correct message for values less than or equal to 100", async function () {
      expect(await ifElse.simpleIf(100)).to.equal("Less than or equal to 100");
      expect(await ifElse.simpleIf(50)).to.equal("Less than or equal to 100");
      expect(await ifElse.simpleIf(0)).to.equal("Less than or equal to 100");
    });
  });

  describe("If-Else", function () {
    it("should return correct message for values greater than 100", async function () {
      expect(await ifElse.ifElse(150)).to.equal("Greater than 100");
    });

    it("should return correct message for values less than or equal to 100", async function () {
      expect(await ifElse.ifElse(100)).to.equal("Less than or equal to 100");
      expect(await ifElse.ifElse(50)).to.equal("Less than or equal to 100");
    });
  });

  describe("If-Else If", function () {
    it("should return correct message for values greater than 100", async function () {
      expect(await ifElse.ifElseIf(150)).to.equal("Greater than 100");
    });

    it("should return correct message for values between 50 and 100", async function () {
      expect(await ifElse.ifElseIf(75)).to.equal("Greater than 50 but less than or equal to 100");
      expect(await ifElse.ifElseIf(100)).to.equal("Greater than 50 but less than or equal to 100");
    });

    it("should return correct message for values between 0 and 50", async function () {
      expect(await ifElse.ifElseIf(25)).to.equal("Greater than 0 but less than or equal to 50");
      expect(await ifElse.ifElseIf(50)).to.equal("Greater than 0 but less than or equal to 50");
    });

    it("should return correct message for zero or negative values", async function () {
      expect(await ifElse.ifElseIf(0)).to.equal("Zero or negative");
    });
  });

  describe("Nested If", function () {
    it("should return correct message for positive values greater than 100", async function () {
      expect(await ifElse.nestedIf(150)).to.equal("Positive and greater than 100");
    });

    it("should return correct message for positive values less than or equal to 100", async function () {
      expect(await ifElse.nestedIf(75)).to.equal("Positive but less than or equal to 100");
      expect(await ifElse.nestedIf(100)).to.equal("Positive but less than or equal to 100");
    });

    it("should return correct message for zero or negative values", async function () {
      expect(await ifElse.nestedIf(0)).to.equal("Zero or negative");
    });
  });

  describe("Conditional Assignment", function () {
    it("should double value when greater than 100", async function () {
      expect(await ifElse.conditionalAssignment(150)).to.equal(300);
    });

    it("should halve value when less than or equal to 100", async function () {
      expect(await ifElse.conditionalAssignment(100)).to.equal(50);
      expect(await ifElse.conditionalAssignment(50)).to.equal(25);
    });
  });

  describe("Ternary Operator", function () {
    it("should return 'Large' for values greater than 100", async function () {
      expect(await ifElse.ternaryOperator(150)).to.equal("Large");
    });

    it("should return 'Small' for values less than or equal to 100", async function () {
      expect(await ifElse.ternaryOperator(100)).to.equal("Small");
      expect(await ifElse.ternaryOperator(50)).to.equal("Small");
    });
  });

  describe("Complex Conditional", function () {
    it("should return correct message for large even numbers", async function () {
      expect(await ifElse.complexConditional(150)).to.equal("Large even number");
    });

    it("should return correct message for large odd numbers", async function () {
      expect(await ifElse.complexConditional(151)).to.equal("Large odd number");
    });

    it("should return correct message for small even numbers", async function () {
      expect(await ifElse.complexConditional(50)).to.equal("Small even number");
    });

    it("should return correct message for small odd numbers", async function () {
      expect(await ifElse.complexConditional(51)).to.equal("Small odd number");
    });
  });

  describe("State-based Conditional", function () {
    it("should return correct message based on state value", async function () {
      expect(await ifElse.stateBasedConditional()).to.equal("State value is small");
    });
  });
}); 