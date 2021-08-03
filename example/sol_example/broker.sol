pragma solidity >=0.5.6;
pragma experimental ABIEncoderV2;


contract Broker {
    function interchain(string memory serviceid, string memory funcs, string memory args, string memory argsCb, string memory argsRb) public{
        bytes memory input = abi.encode(serviceid, funcs,args, argsCb, argsRb);
        uint256 len = input.length;
        assembly {
            let memPtr := mload(0x40)
            let result := call(gas(), 0xC8, 0, add(input, 0x20), len , memPtr, 0x20)
        }
    }

}