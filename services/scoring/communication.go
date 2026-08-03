package scoring

// computeCommunication 计算沟通维度分（SCORING-SPEC 6.4）。
// TASK-040 覆盖 voice 模式：structure_clarity × 0.6 + oral_delivery × 0.4；
// text/mixed 归一化由 TASK-041 扩展（文字模式口语 not_evaluated 不记 0、混合模式按占比合并）。
func (s *Service) computeCommunication(
	_ Input, cps []CoverageAssessment, base DimensionResult,
) DimensionResult {
	sumSC, weightSC := 0, 0
	sumOD, weightOD := 0, 0
	var evidenceIDs []string
	for _, cp := range cps {
		if cp.AnswerStatus != AnswerAnswered && cp.AnswerStatus != AnswerPartial {
			continue
		}
		sc := cp.StructureClarity
		if sc == nil {
			value := interpolatedScore(cp)
			sc = &value
		}
		sumSC += *sc * cp.WeightInDimension
		weightSC += cp.WeightInDimension
		od := cp.OralDelivery
		if od == nil {
			value := interpolatedScore(cp)
			od = &value
		}
		sumOD += *od * cp.WeightInDimension
		weightOD += cp.WeightInDimension
		evidenceIDs = append(evidenceIDs, cp.EvidenceIDs...)
	}
	structureClarity := roundHalfUp(float64(sumSC) / float64(weightSC))
	oralDelivery := roundHalfUp(float64(sumOD) / float64(weightOD))
	score := roundHalfUp(0.6*float64(structureClarity) + 0.4*float64(oralDelivery))
	base.ScoreStatus = StatusScored
	base.Score = &score
	base.Subscores = &CommunicationSubscores{
		StructureClarity: &structureClarity,
		OralDelivery:     &oralDelivery,
	}
	base.EvidenceIDs = uniqueStrings(evidenceIDs)
	return base
}
