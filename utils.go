package main

func (slice CrossTrafficComponentArr) Len() int {
	return len(slice)
}

func (slice CrossTrafficComponentArr) Less(i, j int) bool {
	return slice[i].NextEvent < slice[j].NextEvent
}

func (slice CrossTrafficComponentArr) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
