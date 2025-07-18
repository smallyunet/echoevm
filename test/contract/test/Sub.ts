import {
  time,
  loadFixture,
} from "@nomicfoundation/hardhat-toolbox/network-helpers";
import { anyValue } from "@nomicfoundation/hardhat-chai-matchers/withArgs";
import { expect } from "chai";
import hre from "hardhat";

describe("Sub", function () {
  async function deploySubFixture() {
    const [owner, otherAccount] = await hre.ethers.getSigners();

    const Sub = await hre.ethers.getContractFactory("Sub");
    const sub = await Sub.deploy();

    return { sub, owner, otherAccount };
  }

  describe("Deployment", function () {
    it("Should set the right result", async function () {
      const { sub } = await loadFixture(deploySubFixture);
      const ret = await sub.sub(5, 2);
      expect(ret).to.equal(3);
    });
  });
});
