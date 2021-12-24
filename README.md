<!-- vim-markdown-toc GFM -->

+ [io-hat go](#io-hat-go)
	* [Kaip veikia](#kaip-veikia)
	* [Kaip paruošti Raspberry Pi](#kaip-paruošti-raspberry-pi)
	* [Šaltiniai](#Šaltiniai)

<!-- vim-markdown-toc -->

# io-hat go



## Kaip veikia

Įėjimų statusas publikuojamas periodiškai ir kai pasikeičia reikšmė
- stat/rpi/in1 Payload:1/0

Išėjimų statusas publikuojamas periodiškai ir kai pasikeičia reikšmė
- stat/rpi/out1 Payload:1/0

Išėjimo reikšmė keičiama komandomis
- cmd/rpi/out1/set_on
- cmd/rpi/out1/set_off
- cmd/rpi/out1/set Payload: 1/0 on/off?

## Kaip paruošti Raspberry Pi

- https://olab.lt/protingo-namo-stendas-su-raspberry-pi-4-1/


## Šaltiniai

- Paho Mqtt https://github.com/eclipse/paho.mqtt.golang
- 1-Wire DS18B20 Sensor https://www.waveshare.com/wiki/Raspberry_Pi_Tutorial_Series:_1-Wire_DS18B20_Sensor
- configure GPIO on boot https://www.raspberrypi.org/forums/viewtopic.php?f=117&t=208748
- rpi interfacing library gpiod https://github.com/warthog618/gpiod
- rpi gpio tutorial with interrupts and debounce https://roboticsbackend.com/raspberry-pi-gpio-interrupts-tutorial/
