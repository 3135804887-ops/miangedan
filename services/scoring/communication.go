package scoring

// computeCommunication 计算沟通维度分（SCORING-SPEC 6.4，TASK-041 输入模式归一化）。
// voice：0.6×structure_clarity + 0.4×oral_delivery；
// text：communication = structure_clarity，oral_delivery = not_evaluated（不记 0）；
// mixed：按语音/文字有效证据占比合并，报告标注模式与证据限制。
// 摄像头开关与便利设置不进入任何计算（SC-EC-11/12）。
func (s *Service) computeCommunication(
	in Input, cps []CoverageAssessment, base DimensionResult,
) DimensionResult {
	switch in.InputModeContext.CommunicationMode {
	case ModeText:
		return s.communicationText(cps, base)
	case ModeMixed:
		return s.communicationMixed(in, cps, base)
	default:
		return s.communicationVoice(cps, base)
	}
}

func (s *Service) communicationVoice(cps []CoverageAssessment, base DimensionResult) DimensionResult {
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

// communicationText 文字模式：口语表现 not_evaluated，不记 0、不扣分（SC-EC-09）。
func (s *Service) communicationText(
	cps []CoverageAssessment, base DimensionResult,
) DimensionResult {
	sum, weightSum := 0, 0
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
		sum += *sc * cp.WeightInDimension
		weightSum += cp.WeightInDimension
		evidenceIDs = append(evidenceIDs, cp.EvidenceIDs...)
	}
	structureClarity := roundHalfUp(float64(sum) / float64(weightSum))
	base.ScoreStatus = StatusScored
	base.Score = &structureClarity
	base.Subscores = &CommunicationSubscores{
		StructureClarity: &structureClarity,
		OralDelivery:     "not_evaluated",
	}
	base.EvidenceIDs = uniqueStrings(evidenceIDs)
	return base
}

// communicationMixed 混合模式：按语音/文字有效证据占比合并（SC-EC-10）。
func (s *Service) communicationMixed(
	in Input, cps []CoverageAssessment, base DimensionResult,
) DimensionResult {
	var voiceCPs, textCPs []CoverageAssessment
	for _, cp := range cps {
		if cp.InputMode == ModeText {
			textCPs = append(textCPs, cp)
		} else {
			voiceCPs = append(voiceCPs, cp)
		}
	}
	voiceScore := 0
	if len(voiceCPs) > 0 {
		voiceScore = *s.communicationVoice(voiceCPs, base).Score
	}
	textScore := 0
	if len(textCPs) > 0 {
		textScore = *s.communicationText(textCPs, base).Score
	}
	share := 0.5
	if in.InputModeContext.MixedModeVoiceShare != nil {
		share = *in.InputModeContext.MixedModeVoiceShare
	}
	score := roundHalfUp(share*float64(voiceScore) + (1-share)*float64(textScore))
	// 子项：structure_clarity 为全部有效证据加权；oral_delivery 仅语音证据（无语音则 not_evaluated）。
	overall := s.communicationText(cps, base)
	structureClarity := *overall.Subscores.StructureClarity
	var oral any = "not_evaluated"
	if len(voiceCPs) > 0 {
		voice := s.communicationVoice(voiceCPs, base)
		oral = *voice.Subscores.OralDelivery.(*int)
	}
	base.ScoreStatus = StatusScored
	base.Score = &score
	base.Subscores = &CommunicationSubscores{
		StructureClarity: &structureClarity,
		OralDelivery:     oral,
	}
	base.EvidenceIDs = uniqueStrings(overall.EvidenceIDs)
	return base
}
