import { expect } from "chai";
import { ethers } from "hardhat";
import { BoolType } from "../../typechain-types";

describe("BoolType", function () {
  let boolType: BoolType;

  beforeEach(async function () {
    const BoolTypeFactory = await ethers.getContractFactory("BoolType");
    boolType = await BoolTypeFactory.deploy();
    await boolType.waitForDeployment();
  });

  describe("Boolean Operations", function () {
    it("should initialize with correct boolean values", async function () {
      expect(await boolType.isActive()).to.equal(true);
      expect(await boolType.isPaused()).to.equal(false);
    });

    it("should toggle active status", async function () {
      expect(await boolType.isActive()).to.equal(true);
      
      await boolType.toggleActive();
      expect(await boolType.isActive()).to.equal(false);
      
      await boolType.toggleActive();
      expect(await boolType.isActive()).to.equal(true);
    });

    it("should set active status", async function () {
      await boolType.setActive(false);
      expect(await boolType.isActive()).to.equal(false);
      
      await boolType.setActive(true);
      expect(await boolType.isActive()).to.equal(true);
    });

    it("should return active status", async function () {
      expect(await boolType.getActiveStatus()).to.equal(true);
    });
  });

  describe("Logical Operations", function () {
    it("should perform logical AND", async function () {
      expect(await boolType.logicalAnd(true, true)).to.equal(true);
      expect(await boolType.logicalAnd(true, false)).to.equal(false);
      expect(await boolType.logicalAnd(false, true)).to.equal(false);
      expect(await boolType.logicalAnd(false, false)).to.equal(false);
    });

    it("should perform logical OR", async function () {
      expect(await boolType.logicalOr(true, true)).to.equal(true);
      expect(await boolType.logicalOr(true, false)).to.equal(true);
      expect(await boolType.logicalOr(false, true)).to.equal(true);
      expect(await boolType.logicalOr(false, false)).to.equal(false);
    });

    it("should perform logical NOT", async function () {
      expect(await boolType.logicalNot(true)).to.equal(false);
      expect(await boolType.logicalNot(false)).to.equal(true);
    });
  });
}); 