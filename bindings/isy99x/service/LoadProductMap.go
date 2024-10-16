package service

import (
	"os"
	"strings"
)

const embeddedProductListCsv = `
Dev Cat,Sub Cat,Device Category Name,Model,Product Name
0x00,0x04,General Controllers,2430,ControLinc
0x00,0x05,General Controllers,2440,RemoteLinc
0x00,0x06,General Controllers,2830,ICON Tabletop Controller
0x00,0x08,General Controllers,N/A,EZBridge/EZServer
0x00,0x09,General Controllers,2442,SignaLinc RF Signal Enhancer
0x00,0x0A,General Controllers,N/A,Poolux LCD Controller
0x00,0x0B,General Controllers,2443,Access Point (Wireless Phase Coupler)
0x00,0x0C,General Controllers,12005,IES Color Touchscreen
0x00,0x0E,General Controllers,2440EZ,RemoteLinc EZ
0x00,0x10,General Controllers,2444A2xx4,"RemoteLinc 2 Keypad, 4 Scene"
0x00,0x11,General Controllers,2444A3xx,RemoteLinc 2 Switch
0x00,0x12,General Controllers,2444A2xx8,"RemoteLinc 2 Keypad, 8 Scene"
0x00,0x13,General Controllers,2993-222,INSTEON Diagnostics Keypad
0x00,0x14,General Controllers,2342-432,INSTEON Mini Remote - 4 Scene (869 MHz)
0x00,0x15,General Controllers,2342-442,INSTEON Mini Remote - Switch (869 MHz)
0x00,0x16,General Controllers,2342-422,INSTEON Mini Remote - 8 Scene (869 MHz)
0x00,0x17,General Controllers,2342-532,INSTEON Mini Remote - 4 Scene (921 MHz)
0x00,0x19,General Controllers,2342-542,INSTEON Mini Remote - Switch (921 MHz)
0x00,0x1A,General Controllers,2342-222,INSTEON Mini Remote - 8 Scene (915 MHz)
0x00,0x1B,General Controllers,2342-232,INSTEON Mini Remote - 4 Scene (915 MHz)
0x00,0x1C,General Controllers,2342-242,INSTEON Mini Remote - Switch (915 MHz)
0x00,0x1D,General Controllers,2992-222,Range Extender
0x01,0x00,Dimmable Lighting Control,2456D3,LampLinc 3-Pin
0x01,0x01,Dimmable Lighting Control,2476D,SwitchLinc Dimmer
0x01,0x02,Dimmable Lighting Control,2475D,In-LineLinc Dimmer
0x01,0x03,Dimmable Lighting Control,2876DB,ICON Dimmer Switch
0x01,0x04,Dimmable Lighting Control,2476DH,SwitchLinc Dimmer (High Wattage)
0x01,0x05,Dimmable Lighting Control,2484DWH8,Keypad Countdown Timer w/ Dimmer
0x01,0x06,Dimmable Lighting Control,2456D2,LampLinc Dimmer (2-Pin)
0x01,0x07,Dimmable Lighting Control,2856D2B,ICON LampLinc
0x01,0x09,Dimmable Lighting Control,2486D,KeypadLinc Dimmer
0x01,0x0A,Dimmable Lighting Control,2886D,Icon In-Wall Controller
0x01,0x0B,Dimmable Lighting Control,2632-422,"INSTEON Dimmer Module, France (869 MHz)"
0x01,0x0C,Dimmable Lighting Control,2486DWH8,KeypadLinc Dimmer
0x01,0x0D,Dimmable Lighting Control,2454D,SocketLinc
0x01,0x0E,Dimmable Lighting Control,2457D2,LampLinc (Dual-Band)
0x01,0x0F,Dimmable Lighting Control,2632-432,"INSTEON Dimmer Module, Germany (869 MHz)"
0x01,0x11,Dimmable Lighting Control,2632-442,"INSTEON Dimmer Module, UK (869 MHz)"
0x01,0x12,Dimmable Lighting Control,2632-522,"INSTEON Dimmer Module, Aus/NZ (921 MHz)"
0x01,0x17,Dimmable Lighting Control,2466D,ToggleLinc Dimmer
0x01,0x18,Dimmable Lighting Control,2474D,Icon SwitchLinc Dimmer Inline Companion
0x01,0x19,Dimmable Lighting Control,2476D,SwitchLinc Dimmer [with beeper]
0x01,0x1A,Dimmable Lighting Control,2475D,In-LineLinc Dimmer [with beeper]
0x01,0x1B,Dimmable Lighting Control,2486DWH6,KeypadLinc Dimmer
0x01,0x1C,Dimmable Lighting Control,2486DWH8,KeypadLinc Dimmer
0x01,0x1D,Dimmable Lighting Control,2476DH,SwitchLinc Dimmer (High Wattage)[beeper]
0x01,0x1E,Dimmable Lighting Control,2876DB,ICON Switch Dimmer
0x01,0x1F,Dimmable Lighting Control,2466Dx,ToggleLinc Dimmer [with beeper]
0x01,0x20,Dimmable Lighting Control,2477D,SwitchLinc Dimmer (Dual-Band)
0x01,0x21,Dimmable Lighting Control,2472D,OutletLinc Dimmer (Dual-Band)
0x01,0x22,Dimmable Lighting Control,2457D2X,LampLinc
0x01,0x23,Dimmable Lighting Control,2457D2EZ,LampLinc Dual-Band EZ
0x01,0x24,Dimmable Lighting Control,2474DWH,SwitchLinc 2-Wire Dimmer (RF)
0x01,0x25,Dimmable Lighting Control,2475DA2,In-LineLinc 0-10VDC Dimmer/Dual-SwitchDB
0x01,0x2D,Dimmable Lighting Control,2477DH,SwitchLinc-Dimmer Dual-Band 1000W
0x01,0x2E,Dimmable Lighting Control,2475F,FanLinc
0x01,0x30,Dimmable Lighting Control,2476D,SwitchLinc Dimmer
0x01,0x31,Dimmable Lighting Control,2478D,SwitchLinc Dimmer 240V-50/60Hz Dual-Band
0x01,0x32,Dimmable Lighting Control,2475DA1,In-LineLinc Dimmer (Dual Band)
0x01,0x34,Dimmable Lighting Control,2452-222,INSTEON DIN Rail Dimmer (915 MHz)
0x01,0x35,Dimmable Lighting Control,2442-222,INSTEON Micro Dimmer (915 MHz)
0x01,0x36,Dimmable Lighting Control,2452-422,INSTEON DIN Rail Dimmer (869 MHz)
0x01,0x37,Dimmable Lighting Control,2452-522,INSTEON DIN Rail Dimmer (921 MHz)
0x01,0x38,Dimmable Lighting Control,2442-422,INSTEON Micro Dimmer (869 MHz)
0x01,0x39,Dimmable Lighting Control,2442-522,INSTEON Micro Dimmer (921 MHz)
0x01,0x3A,Dimmable Lighting Control,2672-222,LED Bulb 240V (915 MHz) - Screw-in Base
0x01,0x41,Dimmable Lighting Control,2334-222,"Keypad Dimmer Dual-Band, 8 Button"
0x01,0x42,Dimmable Lighting Control,2334-232,"Keypad Dimmer Dual-Band, 6 Button"
0x01,0x49,Dimmable Lighting Control,2674-222,LED Bulb PAR38 US/Can - Screw-in Base
0x01,0x4A,Dimmable Lighting Control,2674-422,LED Bulb PAR38 Europe - Screw-in Base
0x01,0x4B,Dimmable Lighting Control,2674-522,LED Bulb PAR38 Aus/NZ - Screw-in Base
0x02,0x06,Switched Lighting Control,2456S3E,Outdoor ApplianceLinc
0x02,0x07,Switched Lighting Control,2456S3T,TimerLinc
0x02,0x08,Switched Lighting Control,2473S,OutletLinc
0x02,0x09,Switched Lighting Control,2456S3,ApplianceLinc (3-Pin)
0x02,0x0A,Switched Lighting Control,2476S,SwitchLinc Relay
0x02,0x0B,Switched Lighting Control,2876S,ICON On/Off Switch
0x02,0x0C,Switched Lighting Control,2856S3,Icon Appliance Module
0x02,0x0D,Switched Lighting Control,2466S,ToggleLinc Relay
0x02,0x0E,Switched Lighting Control,2476ST,SwitchLinc Relay Countdown Timer
0x02,0x0F,Switched Lighting Control,2486SWH6,KeypadLinc On/Off
0x02,0x10,Switched Lighting Control,2475S,In-LineLinc Relay
0x02,0x12,Switched Lighting Control,2474 S/D,ICON In-lineLinc Relay Companion
0x02,0x14,Switched Lighting Control,2475S2,In-LineLinc Relay with Sense
0x02,0x15,Switched Lighting Control,2476SS,SwitchLinc Relay with Sense
0x02,0x16,Switched Lighting Control,2876S,ICON On/Off Switch (25 max links)
0x02,0x17,Switched Lighting Control,2856S3B,ICON Appliance Module
0x02,0x18,Switched Lighting Control,2494S220,SwitchLinc 220V Relay
0x02,0x19,Switched Lighting Control,2494S220,SwitchLinc 220V Relay [with beeper]
0x02,0x1A,Switched Lighting Control,2466Sx,ToggleLinc Relay [with Beeper]
0x02,0x1C,Switched Lighting Control,2476S,SwitchLinc Relay
0x02,0x1E,Switched Lighting Control,2487S,KeypadLinc On/Off (Dual-Band)
0x02,0x1F,Switched Lighting Control,2475SDB,In-LineLinc On/Off (Dual-Band)
0x02,0x25,Switched Lighting Control,2484SWH8,KeypadLinc 8-Button Countdown On/Off Switch Timer
0x02,0x29,Switched Lighting Control,2476ST,SwitchLinc Relay Countdown Timer
0x02,0x2A,Switched Lighting Control,2477S,SwitchLinc Relay (Dual-Band)
0x02,0x2C,Switched Lighting Control,2487S,"KeypadLinc On/Off (Dual-Band,50/60 Hz)"
0x02,0x2D,Switched Lighting Control,2633-422,"INSTEON On/Off Module, France (869 MHz)"
0x02,0x2E,Switched Lighting Control,2453-222,INSTEON DIN Rail On/Off (915 MHz)
0x02,0x2F,Switched Lighting Control,2443-222,INSTEON Micro On/Off (915 MHz)
0x02,0x30,Switched Lighting Control,2633-432,"INSTEON On/Off Module, Germany (869 MHz)"
0x02,0x31,Switched Lighting Control,2443-422,INSTEON Micro On/Off (869 MHz)
0x02,0x32,Switched Lighting Control,2443-522,INSTEON Micro On/Off (921 MHz)
0x02,0x33,Switched Lighting Control,2453-422,INSTEON DIN Rail On/Off (869 MHz)
0x02,0x34,Switched Lighting Control,2453-522,INSTEON DIN Rail On/Off (921 MHz)
0x02,0x35,Switched Lighting Control,2633-442,"INSTEON On/Off Module, UK (869 MHz)"
0x02,0x36,Switched Lighting Control,2633-522,"INSTEON On/Off Module, Aus/NZ (921 MHz)"
0x02,0x37,Switched Lighting Control,2635-222,"INSTEON On/Off Module, US (915 MHz)"
0x02,0x38,Switched Lighting Control,2634-222,On/Off Outdoor Module (Dual-Band)
0x03,0x01,Network Bridges,2414S,PowerLinc Serial Controller
0x03,0x02,Network Bridges,2414U,PowerLinc USB Controller
0x03,0x03,Network Bridges,2814S,ICON PowerLinc Serial
0x03,0x04,Network Bridges,2814U,ICON PowerLinc USB
0x03,0x05,Network Bridges,2412S,PowerLinc Serial Modem
0x03,0x06,Network Bridges,2411R,IRLinc Receiver
0x03,0x07,Network Bridges,2411T,IRLinc Transmitter
0x03,0x0A,Network Bridges,2410S,SeriaLinc - INSTEON to RS232
0x03,0x0B,Network Bridges,2412U,PowerLinc USB Modem
0x03,0x0D,Network Bridges,N/A,SimpleHomeNet EZX10RF
0x03,0x0E,Network Bridges,N/A,X10 TW-523/PSC05 Translator
0x03,0x0F,Network Bridges,EZX10IR,EZX10IR X10 IR Receiver
0x03,0x10,Network Bridges,2412N,SmartLinc
0x03,0x11,Network Bridges,2413S,PowerLinc Serial Modem (Dual Band)
0x03,0x13,Network Bridges,2412UH,PowerLinc USB Modem for HouseLinc
0x03,0x14,Network Bridges,2412SH,PowerLinc Serial Modem for HouseLinc
0x03,0x15,Network Bridges,2413U,PowerLinc USB Modem (Dual Band)
0x03,0x16,Network Bridges,N/A,Compacta-SrvrBee INSTEON/ZigBee/X10 Gtwy
0x03,0x17,Network Bridges,N/A,Compacta-Serial/ZigBee daughter card PLM
0x03,0x19,Network Bridges,2413SH,PowerLinc Serial Modem for HL(Dual Band)
0x03,0x1A,Network Bridges,2413UH,PowerLinc USB Modem for HL (Dual Band)
0x03,0x1B,Network Bridges,2423A4,iGateway
0x03,0x1D,Network Bridges,,PowerLinc Modem Serial w/o EEPROM(w/ RF)
0x03,0x1E,Network Bridges,2412S,PowerLincModemSerial w/o EEPROM(w/o RF)
0x03,0x1F,Network Bridges,2448A7,USB Adapter
0x03,0x20,Network Bridges,2448A7,USB Adapter
0x03,0x21,Network Bridges,2448A7H,Portable USB Adapter for HouseLinc
0x03,0x23,Network Bridges,2448A7H,Portable USB Adapter for HouseLinc
0x03,0x24,Network Bridges,2448A7T,TouchLinc
0x03,0x27,Network Bridges,2448A7T,TouchLinc
0x03,0x2B,Network Bridges,2242-222,INSTEON Hub (915 MHz) - no RF
0x03,0x2E,Network Bridges,2242-422,INSTEON Hub (EU - 869 MHz)
0x03,0x2F,Network Bridges,2242-522,INSTEON Hub (921 MHz)
0x03,0x30,Network Bridges,2242-442,INSTEON Hub (UK - 869 MHz)
0x03,0x31,Network Bridges,2242-232,INSTEON Hub (Plug-In Version)
0x03,0x37,Network Bridges,2242-222,INSTEON Hub (915 MHz) - RF
0x03,0x38,Network Bridges,N/A,Revolv Hub
0x04,0x00,Irrigation Control,31270,Compacta EZRain Sprinkler Controller
0x05,0x00,Climate Control,2670IAQ- 80,Broan SMSC080 Exhaust Fan (no beeper)
0x05,0x01,Climate Control,N/A,Compacta EZTherm
0x05,0x02,Climate Control,2670IAQ- 110,Broan SMSC110 Exhaust Fan (no beeper)
0x05,0x03,Climate Control,2441V,Thermostat Adapter
0x05,0x04,Climate Control,N/A,Compacta EZThermx Thermostat
0x05,0x05,Climate Control,N/A,"Broan, Venmar, BEST Rangehoods"
0x05,0x06,Climate Control,N/A,Broan SmartSense Make-up Damper
0x05,0x0A,Climate Control,2441ZTH,INSTEON Wireless Thermostat (915 MHz)
0x05,0x0B,Climate Control,2441TH,INSTEON Thermostat (915 MHz)
0x05,0x0E,Climate Control,2491TxE,Integrated Remote Control Thermostat
0x05,0x0F,Climate Control,2732-422,INSTEON Thermostat (869 MHz)
0x05,0x10,Climate Control,2732-522,INSTEON Thermostat (921 MHz)
0x05,0x11,Climate Control,2732-432,INSTEON Zone Thermostat (869 MHz)
0x05,0x12,Climate Control,2732-532,INSTEON Zone Thermostat (921 MHz)
0x06,0x00,Pool and Spa Control,N/A,Compacta EZPool
0x06,0x01,Pool and Spa Control,N/A,Low-end pool controller
0x06,0x02,Pool and Spa Control,N/A,Mid-Range pool controller
0x06,0x03,Pool and Spa Control,N/A,Next Generation pool controller
0x07,0x00,Sensors and Actuators,2450,I/OLinc
0x07,0x01,Sensors and Actuators,N/A,Compacta EZSns1W Sensor Interface Module
0x07,0x02,Sensors and Actuators,N/A,Compacta EZIO8T I/O Module
0x07,0x03,Sensors and Actuators,31274,Compacta EZIO2X4 #5010D
0x07,0x04,Sensors and Actuators,N/A,Compacta EZIO8SA I/O Module
0x07,0x05,Sensors and Actuators,31275,Compacta EZSnsRF RcvrIntrfc Dakota Alert
0x07,0x06,Sensors and Actuators,N/A,Compacta EZISnsRf SensorInterface Module
0x07,0x07,Sensors and Actuators,31280,EZIO6I (6 inputs)
0x07,0x08,Sensors and Actuators,31283,EZIO4O (4 relay outputs)
0x07,0x09,Sensors and Actuators,2423A5,SynchroLinc
0x07,0x0D,Sensors and Actuators,2450,I/OLinc 50/60Hz Auto Detect
0x09,0x00,Energy Management,N/A,Compacta EZEnergy
0x09,0x01,Energy Management,N/A,OnSitePro Leak Detector
0x09,0x02,Energy Management,N/A,OnSitePro Control Valve
0x09,0x03,Energy Management,N/A,Energy Inc. TED 5000 Single Phase MTU
0x09,0x04,Energy Management,N/A,Energy Inc. TED 5000 Gateway - USB
0x09,0x05,Energy Management,N/A,Energy Inc. TED 5000 Gateway - Ethernet
0x09,0x06,Energy Management,N/A,Energy Inc. TED 3000 Three Phase MTU
0x09,0x07,Energy Management,2423A1,iMeter Solo
0x09,0x0A,Energy Management,2477SA1,220/240V 30A Load Controller NO (DB)
0x09,0x0B,Energy Management,2477SA2,220/240V 30A Load Controller NC (DB)
0x09,0x0D,Energy Management,2448A2,Energy Display
0x0E,0x00,Window Coverings,318276I,Somfy Drape Controller RF Bridge
0x0E,0x01,Window Coverings,2444-222,INSTEON Micro Open/Remove (915 MHz)
0x0E,0x02,Window Coverings,2444-422,INSTEON Micro Open/Remove (869 MHz)
0x0E,0x03,Window Coverings,2444-522,INSTEON Micro Open/Remove (921 MHz)
0x0F,0x00,Access Control,N/A,Weiland Doors Central Drive and Control
0x0F,0x01,Access Control,N/A,Weiland Doors Secondary Central Drive
0x0F,0x02,Access Control,N/A,Weiland Doors Assist Drive
0x0F,0x03,Access Control,N/A,Weiland Doors Elevation Drive
0x0F,0x04,Access Control,,GarageHawk Garage Unit
0x0F,0x05,Access Control,,GarageHawk Remote Unit
0x0F,0x06,Access Control,2458A1,MorningLinc
0x10,0x00,"Security, Health and Safety",N/A,First Alert ONELink RF to Insteon Bridge
0x10,0x01,"Security, Health and Safety",2842-222,Motion Sensor - US (915 MHz)
0x10,0x02,"Security, Health and Safety",2843-222,INSTEON Open/Remove Sensor (915 MHz)
0x10,0x04,"Security, Health and Safety",2842-422,INSTEON Motion Sensor (869 MHz)
0x10,0x05,"Security, Health and Safety",2842-522,INSTEON Motion Sensor (921 MHz)
0x10,0x06,"Security, Health and Safety",2843-422,INSTEON Open/Remove Sensor (869 MHz)
0x10,0x07,"Security, Health and Safety",2843-522,INSTEON Open/Remove Sensor (921 MHz)
0x10,0x08,"Security, Health and Safety",2852-222,Leak Sensor - US (915 MHz)
0x10,0x0A,"Security, Health and Safety",2982-222,Smoke Bridge
`

type InsteonProduct struct {
	Cat         string
	CatName     string
	Sub         string
	Model       string
	ProductName string
}

// LoadProductMapCSV loads the ISY list of product type and description from
// a CSV file, or use the embedded list if no path is given.
//
// This returns a map of product descriptions keyed by product ID where the
// product ID is in the foramt: {cat}.{subcat}. Both cat and subcat are defined
// with their hex values.
// For example: "0x01.0x04" for a ControlLinc device.
func LoadProductMapCSV(path string) (prodList map[string]InsteonProduct, err error) {
	prodList = make(map[string]InsteonProduct)
	listData := embeddedProductListCsv
	if path != "" {
		raw, err2 := os.ReadFile(path)
		err = err2
		if err != nil {
			return nil, err
		}
		listData = string(raw)
	}
	lines := strings.Split(listData, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) >= 5 {
			item := InsteonProduct{
				Cat:         parts[0],
				CatName:     parts[1],
				Sub:         parts[2],
				Model:       parts[3],
				ProductName: parts[4],
			}
			prodID := parts[0] + "." + parts[1]
			prodList[prodID] = item
		}
	}
	return prodList, err
}
