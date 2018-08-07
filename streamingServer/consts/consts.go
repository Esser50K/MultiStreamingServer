package consts

type Quality string

const (
  // Streaming Quality Consts
  HighQuality Quality = "high"
  LowQuality  Quality = "low"
)

var (
  Qualities []Quality = []Quality{LowQuality, HighQuality}
)
