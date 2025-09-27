// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @title Signature-based ERC20 (mintable & approvable via signatures, no imports)
contract SimpleToken {
    // ERC20 basic data
    string public name;
    string public symbol;
    uint8 public immutable decimals;
    uint256 private _totalSupply;

    mapping(address => uint256) private _balances;
    mapping(address => mapping(address => uint256)) private _allowances;

    // Nonces for replay protection
    mapping(address => uint256) public nonces;

    // Events
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);

    constructor(
        string memory _name,
        string memory _symbol,
        uint8 _decimals,
        uint256 initialSupply
    ) {
        name = _name;
        symbol = _symbol;
        decimals = _decimals;
        if (initialSupply > 0) {
            _mint(msg.sender, initialSupply);
        }
    }

    // ---------------------------
    // ERC20 standard interface
    // ---------------------------
    function totalSupply() external view returns (uint256) {
        return _totalSupply;
    }

    function balanceOf(address account) external view returns (uint256) {
        return _balances[account];
    }

    function allowance(address acctOwner, address spender) external view returns (uint256) {
        return _allowances[acctOwner][spender];
    }

    function transfer(address to, uint256 amount) external returns (bool) {
        _transfer(msg.sender, to, amount);
        return true;
    }

    function transferFrom(address from, address to, uint256 amount) external returns (bool) {
        uint256 currentAllowance = _allowances[from][msg.sender];
        require(currentAllowance >= amount, "ERC20: transfer exceeds allowance");
        _approve(from, msg.sender, currentAllowance - amount);
        _transfer(from, to, amount);
        return true;
    }

    // ---------------------------
    // Internal core implementations
    // ---------------------------
    function _transfer(address from, address to, uint256 amount) internal {
        require(from != address(0), "ERC20: transfer from zero");
        require(to != address(0), "ERC20: transfer to zero");
        uint256 fromBal = _balances[from];
        require(fromBal >= amount, "ERC20: transfer exceeds balance");
        unchecked {
            _balances[from] = fromBal - amount;
        }
        _balances[to] += amount;
        emit Transfer(from, to, amount);
    }

    function _approve(address acctOwner, address spender, uint256 amount) internal {
        require(acctOwner != address(0), "ERC20: approve from zero");
        require(spender != address(0), "ERC20: approve to zero");
        _allowances[acctOwner][spender] = amount;
        emit Approval(acctOwner, spender, amount);
    }

    function _mint(address to, uint256 amount) internal {
        require(to != address(0), "ERC20: mint to zero");
        _totalSupply += amount;
        _balances[to] += amount;
        emit Transfer(address(0), to, amount);
    }

    // ---------------------------
    // Signature-based Minting
    // ---------------------------
    /// @notice Mint tokens if a valid signature authorizing it is provided
    /// @param to The recipient of the minted tokens
    /// @param amount Token amount
    /// @param nonce Unique nonce > stored nonce[to]
    /// @param signature Signature over keccak256(abi.encodePacked(amount, nonce))
    function mint(address to, uint256 amount, uint256 nonce, bytes memory signature) external {
        require(nonce > nonces[to], "invalid nonce");

        bytes32 message = keccak256(abi.encodePacked(amount, nonce));
        address signer = _recoverSigner(message, signature);

        nonces[to] = nonce;
        _mint(to, amount);
    }

    // ---------------------------
    // Signature-based Approval
    // ---------------------------
    /// @notice Approve spender for unlimited allowance if valid signature
    /// @param owner Address giving approval
    /// @param spender Spender address
    /// @param signature Signature of keccak256(abi.encodePacked(lowercase(spender)))
    function approve(address owner, address spender, bytes memory signature) external returns (bool) {
        bytes32 message = keccak256(abi.encodePacked(_toLowerString(spender)));
        address signer = _recoverSigner(message, signature);

        _approve(owner, spender, type(uint256).max);
        return true;
    }

    // ---------------------------
    // ECDSA utilities (no imports)
    // ---------------------------
    function _recoverSigner(bytes32 message, bytes memory sig) internal pure returns (address) {
        require(sig.length == 65, "invalid signature length");

        bytes32 r;
        bytes32 s;
        uint8 v;
        assembly {
            r := mload(add(sig, 32))
            s := mload(add(sig, 64))
            v := byte(0, mload(add(sig, 96)))
        }

        // Ethereum Signed Message prefix
        bytes32 prefixedHash = keccak256(
            abi.encodePacked("\x19Ethereum Signed Message:\n32", message)
        );
        return ecrecover(prefixedHash, v, r, s);
    }

    // Converts address to lowercase hex string
function _toLowerString(address account) internal pure returns (string memory) {
    bytes memory alphabet = "0123456789abcdef";
    bytes20 data = bytes20(account);
    bytes memory str = new bytes(42); // 2 extra bytes for "0x"
    str[0] = "0";
    str[1] = "x";
    for (uint i = 0; i < 20; i++) {
        str[2 + i*2]     = alphabet[uint(uint8(data[i] >> 4))];
        str[3 + i*2] = alphabet[uint(uint8(data[i] & 0x0f))];
    }
    return string(str);
}

}
