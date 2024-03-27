package service

// DigiTwinService service maintains a digital copy of IoT device's description and state.
type DigiTwinService struct {
}

func NewDigiTwinService() *DigiTwinService {
	svc := DigiTwinService{}
	return &svc
}
