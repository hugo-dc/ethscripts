vo_ranges = [
        (0x00, 0x0b), 
        (0x10, 0x1d), 
        (0x20, 0x20), 
        (0x30, 0x3f), 
        (0x40, 0x48), 
        (0x50, 0x5b), 
        #(0x60, 0x6f), 
        #(0x70, 0x7f), 
        (0x80, 0x8f), 
        (0x90, 0x9f), 
        (0xa0, 0xa4), 
        (0xf0, 0xf5), 
        (0xfa, 0xfa), 
        (0xfd, 0xfd), 
        (0xfe, 0xfe), 
        (0xff, 0xff)]

valid_opcodes = ""
for r in vo_ranges:
    for i in range(r[0], r[1] + 1):
        hop = hex(i)[2:]
        if len(hop) < 2:
            hop = '0' + hop
        valid_opcodes += hop

print((len(valid_opcodes)/2)/32)
print(valid_opcodes)


