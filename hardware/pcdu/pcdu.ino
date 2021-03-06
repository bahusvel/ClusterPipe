//#pragma GCC diagnostic ignored "-Wwrite-strings"

#define BITMASK(b) (1 << ((b) % 8))
#define BITSLOT(b) ((b) / 8)
#define BITNSLOTS(nb) ((nb + 8 - 1) / 8)
#define BITSET(a, b) ((a)[BITSLOT(b)] |= BITMASK(b))
#define BITCLEAR(a, b) ((a)[BITSLOT(b)] &= ~BITMASK(b))
#define BITTEST(a, b) ((a)[BITSLOT(b)] & BITMASK(b))

#define CMD_SIZE 100

// Use this to define the mapping between logical units and physical registers
const byte UNIT_MAP[] = {0,1,2,3,4,5,6,7};

#define NUM_UNITS ((int)sizeof(UNIT_MAP))

#define LATCH_PIN 4
#define CLOCK_PIN 3
#define DATA_PIN 5

byte power_set[BITNSLOTS(NUM_UNITS)];

typedef void (*cmdHandler)(char *params);

struct command {
	const char *name;
	cmdHandler handle;
};

void unimplemented(char *args) {
	Serial.println("ERR UNKNOWN COMMAND");
}

void onCmd(char *args) {
	int slot = atoi(args);
	if (!slot || slot > NUM_UNITS) {
		Serial.println("ERR Invalid slot");
		return;
	}
	Serial.print("OK Turning ");
	Serial.print(slot);
	Serial.println(" on");
  BITSET(power_set, UNIT_MAP[slot-1]);
}

void ionCmd(char *args) {
  onCmd(args);
  commitCmd((char *)"ion");
}

void offCmd(char *args) {
	int slot = atoi(args);
	if (!slot || slot > NUM_UNITS) {
		Serial.println("ERR Invalid slot");
		return;
	}
	Serial.print("OK Turning ");
	Serial.print(slot);
	Serial.println(" off");
  BITCLEAR(power_set, UNIT_MAP[slot-1]);
}

void ioffCmd(char *args) {
  offCmd(args);
  commitCmd((char *)"ioff");
}

void testCmd(char *args) {
  int slot = atoi(args);
  if (!slot || slot > NUM_UNITS) {
    Serial.println("ERR Invalid slot");
    return;
  }
  if (BITTEST(power_set, UNIT_MAP[slot-1])) {
    Serial.println("OK ON");
  } else {
    Serial.println("OK OFF");
  }
}

void alloffCmd(char *args) {
  for (int i = 0; i < BITNSLOTS(NUM_UNITS); i++) {
    power_set[i] = 0x0;
  }
  commitCmd((char*)"alloff");
}

void commitCmd(char *args) {
  Serial.println("OK Commiting");
  digitalWrite(LATCH_PIN, LOW);
  for (int i = 0; i < BITNSLOTS(NUM_UNITS); i++) {
    shiftOut(DATA_PIN, CLOCK_PIN, MSBFIRST, power_set[i]); 
  }
  digitalWrite(LATCH_PIN, HIGH);
}

int hasPrefix(const char *str, const char *pre)
{
    size_t lenpre = strlen(pre),
           lenstr = strlen(str);
    return lenstr < lenpre ? false : strncmp(pre, str, lenpre) == 0;
}

void setup() {
  // initialize serial:
  Serial.begin(9600);
  // make the pins outputs:
  pinMode(LED_BUILTIN, OUTPUT);
  pinMode(LATCH_PIN, OUTPUT);
  pinMode(CLOCK_PIN, OUTPUT);
  pinMode(DATA_PIN, OUTPUT);
  commitCmd((char*)"start");
}

command cmdTable[]{
	{"on", onCmd},
  {"ion", ionCmd},
	{"off", offCmd},
  {"ioff", ioffCmd},
  {"commit", commitCmd},
	{"test", testCmd},
	{"alloff", alloffCmd},
};

void rxCommand(char *cmd) {
	for (unsigned int i = 0; i < sizeof(cmdTable)/sizeof(struct command); i++) {
		if (hasPrefix(cmd, cmdTable[i].name)){
			cmdTable[i].handle(cmd + strlen(cmdTable[i].name));
			return;
		}
	}
	unimplemented(cmd);
}

void loop() {
	char cmdbuf[CMD_SIZE] = {0};
	int ctr = 0;
	while (Serial.available() > 0) {
		char inchar = Serial.read();
		if (ctr == CMD_SIZE-1) {
			ctr = 0;
		}
		if (inchar == '\n') {
			cmdbuf[ctr] = '\0';
			rxCommand(cmdbuf);
			ctr = 0;
		} else {
			cmdbuf[ctr++] = inchar;
		}
	}
}
