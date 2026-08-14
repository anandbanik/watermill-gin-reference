#!/usr/bin/env python3
"""
Byte-level Vigenere Cipher for Markdown files
Reads a markdown file's raw bytes, encrypts/decrypts them with a repeating-key
Vigenere cipher (mod 256), and base64-encodes the ciphertext so the output
file only ever contains printable ASCII characters. This preserves every
character in the original markdown (formatting, unicode, whitespace, etc.)
exactly on a round trip.

Usage:
    python VigenereCipher.py encrypt input.md encrypted.txt PUZZLE42
    python VigenereCipher.py decrypt encrypted.txt decrypted.md PUZZLE42
"""

import sys
import base64


def vigenere_bytes(data: bytes, key: bytes, decrypt: bool = False) -> bytes:
    """Apply a repeating-key Vigenere cipher (mod 256) to raw bytes."""
    if not key:
        raise ValueError("Key must contain at least one character.")

    result = bytearray(len(data))
    for i, b in enumerate(data):
        k = key[i % len(key)]
        result[i] = (b - k) % 256 if decrypt else (b + k) % 256
    return bytes(result)


def encrypt_file(input_path, output_path, key):
    with open(input_path, 'rb') as f:
        plaintext = f.read()

    ciphertext = vigenere_bytes(plaintext, key.encode('utf-8'), decrypt=False)
    encoded = base64.b64encode(ciphertext)

    with open(output_path, 'wb') as f:
        f.write(encoded)

    return len(plaintext), len(encoded)


def decrypt_file(input_path, output_path, key):
    with open(input_path, 'rb') as f:
        encoded = f.read()

    ciphertext = base64.b64decode(encoded)
    plaintext = vigenere_bytes(ciphertext, key.encode('utf-8'), decrypt=True)

    with open(output_path, 'wb') as f:
        f.write(plaintext)

    return len(encoded), len(plaintext)


def main():
    if len(sys.argv) != 5:
        print("Usage: python VigenereCipher.py <encrypt|decrypt> <input_file> <output_file> <key>")
        sys.exit(1)

    mode, input_path, output_path, key = sys.argv[1:5]

    if mode not in ("encrypt", "decrypt"):
        print("Mode must be 'encrypt' or 'decrypt'")
        sys.exit(1)

    if mode == "encrypt":
        in_len, out_len = encrypt_file(input_path, output_path, key)
    else:
        in_len, out_len = decrypt_file(input_path, output_path, key)

    print(f"{mode.capitalize()}ed '{input_path}' -> '{output_path}' using key '{key}'")
    print(f"Input size: {in_len} bytes, output size: {out_len} bytes")


if __name__ == "__main__":
    main()
