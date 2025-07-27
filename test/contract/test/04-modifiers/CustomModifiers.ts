import { expect } from "chai";
import { ethers } from "hardhat";

describe("CustomModifiers", function () {
  let customModifiers: any;
  let owner: any;
  let user1: any;

  beforeEach(async function () {
    [owner, user1] = await ethers.getSigners();
    const CustomModifiersFactory = await ethers.getContractFactory("CustomModifiers");
    customModifiers = await CustomModifiersFactory.deploy();
    await customModifiers.waitForDeployment();
  });

  describe("OnlyOwner Modifier", function () {
    it("should allow owner to call functions with onlyOwner modifier", async function () {
      await expect(customModifiers.setValue(100)).to.not.be.reverted;
      expect(await customModifiers.getValue()).to.equal(100);
    });

    it("should revert when non-owner calls functions with onlyOwner modifier", async function () {
      await expect(
        customModifiers.connect(user1).setValue(100)
      ).to.be.revertedWith("Only owner can call this function");
    });

    it("should allow owner to transfer ownership", async function () {
      await customModifiers.transferOwnership(user1.address);
      expect(await customModifiers.owner()).to.equal(user1.address);
    });
  });

  describe("WhenNotPaused Modifier", function () {
    it("should allow calls when contract is not paused", async function () {
      await expect(customModifiers.setValue(100)).to.not.be.reverted;
    });

    it("should revert when contract is paused", async function () {
      await customModifiers.pause();
      await expect(
        customModifiers.setValue(100)
      ).to.be.revertedWith("Contract is paused");
    });
  });

  describe("WhenPaused Modifier", function () {
    it("should allow unpause when contract is paused", async function () {
      await customModifiers.pause();
      await expect(customModifiers.unpause()).to.not.be.reverted;
      expect(await customModifiers.paused()).to.equal(false);
    });

    it("should revert unpause when contract is not paused", async function () {
      await expect(
        customModifiers.unpause()
      ).to.be.revertedWith("Contract is not paused");
    });
  });

  describe("OnlyValue Modifier", function () {
    it("should allow increment when value is high enough", async function () {
      await customModifiers.setValue(20);
      await expect(customModifiers.incrementValue()).to.not.be.reverted;
    });

    it("should revert increment when value is too low", async function () {
      await customModifiers.setValue(5);
      await expect(
        customModifiers.incrementValue()
      ).to.be.revertedWith("Value too low");
    });
  });

  describe("BeforeAndAfter Modifier", function () {
    it("should execute code before and after function", async function () {
      // First set a value that meets the onlyValue(10) requirement
      await customModifiers.setValue(15);
      const initialValue = await customModifiers.getValue();
      await customModifiers.incrementValue();
      const finalValue = await customModifiers.getValue();
      
      // Value should be increased by 7: 1 (before) + 5 (function) + 1 (after)
      expect(finalValue).to.equal(initialValue + 7n);
    });
  });

  describe("Combined Modifiers", function () {
    it("should work with multiple modifiers", async function () {
      await expect(customModifiers.emergencyFunction()).to.not.be.reverted;
      expect(await customModifiers.getValue()).to.equal(0);
    });

    it("should revert with combined modifiers when conditions not met", async function () {
      await customModifiers.pause();
      await expect(
        customModifiers.emergencyFunction()
      ).to.be.revertedWith("Contract is paused");
    });
  });

  describe("State Management", function () {
    it("should correctly manage paused state", async function () {
      expect(await customModifiers.paused()).to.equal(false);
      
      await customModifiers.pause();
      expect(await customModifiers.paused()).to.equal(true);
      
      await customModifiers.unpause();
      expect(await customModifiers.paused()).to.equal(false);
    });

    it("should correctly manage value state", async function () {
      expect(await customModifiers.getValue()).to.equal(0);
      
      await customModifiers.setValue(100);
      expect(await customModifiers.getValue()).to.equal(100);
    });
  });
}); 