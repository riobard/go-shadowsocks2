/*
Package shadowaead implements a simple AEAD-protected secure protocol.

In general, there are two types of connections: stream-oriented and packet-oriented.
Stream-oriented connections (e.g. TCP) assume reliable and orderly delivery of bytes.
Packet-oriented connections (e.g. UDP) assume unreliable and out-of-order delivery of packets,
where each packet is either delivered intact or lost.

An encrypted stream starts with a nonce, followed by any number of encrypted records.
Each encrypted record has the following structure:

    [encrypted payload length]
    [payload length tag]
    [encrypted payload]
    [payload tag]

Payload length is 2-byte unsigned big-endian integer capped at 0x3FFF (16383).
The higher 2 bits are reserved and must be set to zero. The first AEAD encrypt/decrypt
operation uses the nonce at the beginning of the stream. After each encrypt/decrypt operation,
the nonce is incremented by one as if it were an unsigned little-endian integer.


Each encrypted packet transmitted on a packet-oriented connection has the following structure:

    [nonce]
    [encrypted payload]
    [payload tag]

Packets are encrypted/decrypted independently.

In both stream-oriented and packet-oriented connections, length of nonce and tag varies
depending on which AEAD is used. Nonces are assumed to be randomly generated and
of sufficient length (at least 12 bytes).
*/
package shadowaead
