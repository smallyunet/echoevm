import { expect } from "chai";
import { ethers } from "hardhat";

describe("ArrayTypes", function () {
  let arrayTypes: any;

  beforeEach(async function () {
    const ArrayTypesFactory = await ethers.getContractFactory("ArrayTypes");
    arrayTypes = await ArrayTypesFactory.deploy();
    await arrayTypes.waitForDeployment();
  });

  describe("Fixed Arrays", function () {
    it("should initialize fixed array correctly", async function () {
      expect(await arrayTypes.fixedArray(0)).to.equal(1);
      expect(await arrayTypes.fixedArray(1)).to.equal(2);
      expect(await arrayTypes.fixedArray(2)).to.equal(3);
      expect(await arrayTypes.fixedArray(3)).to.equal(4);
      expect(await arrayTypes.fixedArray(4)).to.equal(5);
    });

    it("should get fixed array element", async function () {
      expect(await arrayTypes.getFixedArrayElement(0)).to.equal(1);
      expect(await arrayTypes.getFixedArrayElement(4)).to.equal(5);
    });

    it("should set fixed array element", async function () {
      await arrayTypes.setFixedArrayElement(0, 100);
      expect(await arrayTypes.getFixedArrayElement(0)).to.equal(100);
    });

    it("should revert when accessing out of bounds", async function () {
      await expect(arrayTypes.getFixedArrayElement(5)).to.be.revertedWith("Index out of bounds");
      await expect(arrayTypes.setFixedArrayElement(5, 100)).to.be.revertedWith("Index out of bounds");
    });
  });

  describe("Dynamic Arrays", function () {
    it("should initialize dynamic array correctly", async function () {
      expect(await arrayTypes.getDynamicArrayLength()).to.equal(3);
      expect(await arrayTypes.getDynamicArrayElement(0)).to.equal(10);
      expect(await arrayTypes.getDynamicArrayElement(1)).to.equal(20);
      expect(await arrayTypes.getDynamicArrayElement(2)).to.equal(30);
    });

    it("should add elements to dynamic array", async function () {
      const initialLength = await arrayTypes.getDynamicArrayLength();
      await arrayTypes.addToDynamicArray(40);
      expect(await arrayTypes.getDynamicArrayLength()).to.equal(initialLength + 1n);
      expect(await arrayTypes.getDynamicArrayElement(3)).to.equal(40);
    });

    it("should remove elements from dynamic array", async function () {
      const initialLength = await arrayTypes.getDynamicArrayLength();
      await arrayTypes.removeFromDynamicArray();
      expect(await arrayTypes.getDynamicArrayLength()).to.equal(initialLength - 1n);
    });

    it("should set dynamic array element", async function () {
      await arrayTypes.setDynamicArrayElement(1, 999);
      expect(await arrayTypes.getDynamicArrayElement(1)).to.equal(999);
    });

    it("should delete array element", async function () {
      await arrayTypes.deleteArrayElement(1);
      expect(await arrayTypes.getDynamicArrayElement(1)).to.equal(0);
    });

    it("should get array slice", async function () {
      const slice = await arrayTypes.getArraySlice(0, 2);
      expect(slice.length).to.equal(2);
      expect(slice[0]).to.equal(10);
      expect(slice[1]).to.equal(20);
    });
  });

  describe("Multi-dimensional Arrays", function () {
    it("should get multi-array element", async function () {
      expect(await arrayTypes.getMultiArrayElement(0, 0)).to.equal(1);
      expect(await arrayTypes.getMultiArrayElement(0, 1)).to.equal(2);
      expect(await arrayTypes.getMultiArrayElement(1, 0)).to.equal(3);
      expect(await arrayTypes.getMultiArrayElement(1, 1)).to.equal(4);
    });

    it("should set multi-array element", async function () {
      await arrayTypes.setMultiArrayElement(0, 0, 100);
      expect(await arrayTypes.getMultiArrayElement(0, 0)).to.equal(100);
    });

    it("should revert when accessing out of bounds", async function () {
      await expect(arrayTypes.getMultiArrayElement(3, 0)).to.be.revertedWith("Index out of bounds");
      await expect(arrayTypes.setMultiArrayElement(3, 0, 100)).to.be.revertedWith("Index out of bounds");
    });
  });

  describe("String Arrays", function () {
    it("should initialize string array correctly", async function () {
      expect(await arrayTypes.getStringArrayLength()).to.equal(2);
      expect(await arrayTypes.getStringArrayElement(0)).to.equal("Hello");
      expect(await arrayTypes.getStringArrayElement(1)).to.equal("World");
    });

    it("should add string to array", async function () {
      const initialLength = await arrayTypes.getStringArrayLength();
      await arrayTypes.addToStringArray("Test");
      expect(await arrayTypes.getStringArrayLength()).to.equal(initialLength + 1n);
      expect(await arrayTypes.getStringArrayElement(2)).to.equal("Test");
    });
  });
}); 