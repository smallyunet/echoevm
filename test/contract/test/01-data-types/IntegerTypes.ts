import { expect } from "chai";
import { ethers } from "hardhat";

describe("IntegerTypes", function () {
  let integerTypes: any;

  beforeEach(async function () {
    const IntegerTypesFactory = await ethers.getContractFactory("IntegerTypes");
    integerTypes = await IntegerTypesFactory.deploy();
    await integerTypes.waitForDeployment();
  });

  describe("Integer Values", function () {
    it("should initialize with correct integer values", async function () {
      expect(await integerTypes.uint8Value()).to.equal(255);
      expect(await integerTypes.uint16Value()).to.equal(65535);
      expect(await integerTypes.uint256Value()).to.equal(123456789);
      expect(await integerTypes.int8Value()).to.equal(-128);
      expect(await integerTypes.int16Value()).to.equal(-32768);
      expect(await integerTypes.int256Value()).to.equal(-123456789);
    });
  });

  describe("Arithmetic Operations", function () {
    it("should perform addition", async function () {
      expect(await integerTypes.add(5, 3)).to.equal(8);
      expect(await integerTypes.add(100, 200)).to.equal(300);
    });

    it("should perform subtraction", async function () {
      expect(await integerTypes.subtract(10, 3)).to.equal(7);
      expect(await integerTypes.subtract(10, 5)).to.equal(5);
    });

    it("should revert on subtraction underflow", async function () {
      await expect(integerTypes.subtract(5, 10)).to.be.revertedWithPanic(0x11);
    });

    it("should perform multiplication", async function () {
      expect(await integerTypes.multiply(5, 3)).to.equal(15);
      expect(await integerTypes.multiply(10, 10)).to.equal(100);
    });

    it("should perform division", async function () {
      expect(await integerTypes.divide(10, 2)).to.equal(5);
      expect(await integerTypes.divide(7, 3)).to.equal(2); // Integer division
    });

    it("should perform modulo", async function () {
      expect(await integerTypes.modulo(10, 3)).to.equal(1);
      expect(await integerTypes.modulo(15, 4)).to.equal(3);
    });

    it("should perform power", async function () {
      expect(await integerTypes.power(2, 3)).to.equal(8);
      expect(await integerTypes.power(5, 2)).to.equal(25);
    });
  });

  describe("Increment and Decrement", function () {
    it("should increment values", async function () {
      expect(await integerTypes.increment(5)).to.equal(6);
      expect(await integerTypes.increment(0)).to.equal(1);
    });

    it("should decrement values", async function () {
      expect(await integerTypes.decrement(5)).to.equal(4);
      expect(await integerTypes.decrement(1)).to.equal(0);
    });

    it("should revert on decrement underflow", async function () {
      await expect(integerTypes.decrement(0)).to.be.revertedWithPanic(0x11);
    });
  });
}); 