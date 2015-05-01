# bytering

Go implementation of cyclic slice for bytes, keeps last X bytes and forget the rest.

## Example

    buf := NewByteRing(10)
    buf.Write([]byte("Tutaj"))
    buf.Write([]byte("jest"))
    buf.Write([]byte("tekst."))
    d = make([]byte, 10)
    buf.WriteTo(d) // d will contain "jesttekst."
