package Protocol

type Option struct {
	MagicNumber int
	CodecType   SerializerEnum // supporting only the gob for now
}

const DefaultMagicNumber = 0x3bef5c

var DefaultOption = Option{
	MagicNumber: DefaultMagicNumber,
	CodecType:   GobType,
}
