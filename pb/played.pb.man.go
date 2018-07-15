package pb

func (g *GetPlayedResponse) Len() int {
	return len(g.Games)
}

func (g *GetPlayedResponse) Swap(i, j int) {
	g.Games[i], g.Games[j] = g.Games[j], g.Games[i]
}

func (g *GetPlayedResponse) Less(i, j int) bool {
	return g.Games[i].Dur >= g.Games[i].Dur
}
