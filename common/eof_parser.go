package common

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type EOFObject struct {
	Version      int64
	Types        [][]int64
	CodeSections []string
	Data         string
}

const (
	cOldCodeId = "01"
	cOldDataId = "02"
	cOldTypeId = "03"
	cTypeId    = "01"
	cCodeId    = "02"
	cDataId    = "03"
)

func NewEOFObject() EOFObject {
	return EOFObject{
		Version:      int64(1),
		CodeSections: make([]string, 0),
		Types:        make([][]int64, 0),
		Data:         "",
	}
}

func (eof *EOFObject) CodeNew(withTypes bool) string {
	return eof.Code(false, withTypes)
}

func (eof *EOFObject) Code(old bool, withTypes bool) string {
	eof_code := "ef00"

	typeId := cTypeId
	codeId := cCodeId
	dataId := cDataId

	if old {
		typeId = cOldTypeId
		codeId = cOldCodeId
		dataId = cOldDataId
	}

	versionHex := fmt.Sprintf("%02x", eof.Version)
	typesHeader := ""
	if len(eof.Types) > 0 {
		typesLengthHex := ""
		if old {
			typesLengthHex = fmt.Sprintf("%04x", len(eof.Types)*2)
		} else {
			typesLengthHex = fmt.Sprintf("%04x", len(eof.Types)*4)
		}
		typesHeader = typeId + typesLengthHex
	}

	codeHeaders := ""
	oldCodeHeaders := ""
	codeLengths := ""
	numCodeSections := 0
	for _, c := range eof.CodeSections {
		codeLengthHex := fmt.Sprintf("%04x", len(c)/2)
		codeLengths = codeLengths + codeLengthHex
		oldCodeHeaders = oldCodeHeaders + codeId + codeLengthHex
		numCodeSections += 1
	}
	numCodeSectionsHex := fmt.Sprintf("%04x", numCodeSections)
	codeHeaders = codeId + numCodeSectionsHex + codeLengths

	dataHeader := ""
	dataLengthHex := fmt.Sprintf("%04x", len(eof.Data)/2)
	dataHeader = dataHeader + dataId + dataLengthHex

	terminator := "00"

	typeContents := ""
	for i, t := range eof.Types {
		inputsHex := fmt.Sprintf("%02x", t[0])
		outputsHex := fmt.Sprintf("%02x", t[1])

		if old {
			typeContents = typeContents + inputsHex + outputsHex
		} else {
			maxStackHeight := calculateMaxStack(i, eof.CodeSections[i], eof.Types)
			maxStackHeightHex := fmt.Sprintf("%04x", maxStackHeight)
			typeContents = typeContents + inputsHex + outputsHex + maxStackHeightHex
		}
	}

	codeContents := ""
	for _, c := range eof.CodeSections {
		codeContents = codeContents + c
	}

	if withTypes == false && len(eof.Types) == 1 && old == true {
		if len(eof.Data) > 0 {
			eof_code = eof_code + versionHex + oldCodeHeaders + dataHeader + terminator + codeContents + eof.Data
		} else {
			eof_code = eof_code + versionHex + oldCodeHeaders + terminator + codeContents
		}
	} else {
		fmt.Println("HEADER\n--------")
		fmt.Println("magic:", eof_code)
		fmt.Println("version:", versionHex)
		fmt.Println("types:", typesHeader)
		fmt.Println("codeHeaders:", codeHeaders)
		fmt.Println("dataHeader:", dataHeader)
		fmt.Println("terminator:", terminator)
		fmt.Println("BODY\n--------")
		fmt.Println("typesSection:", typeContents)
		fmt.Println("codeSection:", codeContents)
		fmt.Println("dataSection:", eof.Data)
		eof_code = eof_code + versionHex + typesHeader + codeHeaders + dataHeader + terminator + typeContents + codeContents + eof.Data
	}
	return eof_code
}

func (eof *EOFObject) AddData(dt string) {
	eof.Data = dt
}

func (eof *EOFObject) AddCode(code string) {
	eof.AddCodeWithType(code, []int64{0, 0})
}

func (eof *EOFObject) AddCodeWithType(code string, codeType []int64) {
	// Add default type for existing code
	if len(eof.CodeSections) > 0 && len(eof.Types) == 0 {
		eof.Types = append(eof.Types, []int64{0, 0})
	}

	// Add default type for new code section
	eof.Types = append(eof.Types, codeType)

	// Add Code
	eof.CodeSections = append(eof.CodeSections, code)
}

func (eof *EOFObject) AddDefaultType() bool {
	if len(eof.Types) == 0 {
		eof.Types = append(eof.Types, []int64{0, 0})
		return true
	} else {
		return false
	}
}

func calculateMaxStack(funcId int, code string, types [][]int64) int64 {
	stackHeights := map[int64]int64{}
	startStackHeight := types[funcId][0]
	maxStackHeight := startStackHeight
	worklist := [][]int64{{0, startStackHeight}}

	opCodes := GetOpcodesByNumber()

	for {
		ix := len(worklist) - 1
		res := worklist[ix]
		pos := res[0]
		stackHeight := res[1]
		worklist = worklist[:ix]

	outer:
		for int(pos*2) < len(code) {
			if pos < 0 {
				fmt.Println("Position out of bounds: ", pos)
				break
			}
			op, _ := strconv.ParseInt(code[pos*2:pos*2+2], 16, 64)
			opCode := opCodes[int(op)]
			fmt.Println(pos, opCode)
			if exp, ok := stackHeights[pos]; ok {
				if stackHeight != exp {
					fmt.Println("stackHeight:", stackHeight, "exp:", exp, "at pos", pos)
					fmt.Println("Error: stack height mismatch for different paths")
					break
				} else {
					break
				}
			} else {
				stackHeights[pos] = stackHeight
			}

			stackHeightRequired := int64(opCode.StackInput)
			stackHeightChange := int64(opCode.StackOutput - opCode.StackInput)

			if stackHeightRequired > stackHeight {
				fmt.Println("stackHeightRequired:", stackHeightRequired, "stackHeight:", stackHeight)
				fmt.Println("stack underflow")
				break
			}

			if (1024 + stackHeightChange) < stackHeight {
				fmt.Println("stack overflow")
				break
			}

			stackHeight += stackHeightChange
			switch {
			case opCode.Name == "CALLF":
				if int(pos*2+6) > len(code) {
					fmt.Println("truncated CALLF")
					break
				}
				calledFuncId, _ := strconv.ParseInt(code[pos*2+2:pos*2+6], 16, 64)
				if int(calledFuncId) >= len(types) {
					fmt.Println("invalid function id")
					break
				}
				stackHeightRequired += int64(types[calledFuncId][0])
				stackHeightChange += int64(types[calledFuncId][1] - types[calledFuncId][0])

				if stackHeightRequired > stackHeight {
					fmt.Println("stack underflow")
					break
				}

				if (1024 + stackHeightChange) < stackHeight {
					fmt.Println("stack overflow")
					break
				}
				stackHeight += stackHeightChange
				pos += 3
			case opCode.Name == "RETF":
				if int64(types[funcId][1]) != stackHeight {
					fmt.Printf("Wrong number of outputs (want:%d, got: %d)", types[funcId][1], stackHeight)
				}
				break outer
			case opCode.Name == "RJUMP":
				if int(pos)*2+6 > len(code) {
					fmt.Println("Error: Truncted RJUMP")
					break
				}
				offset, _ := strconv.ParseInt(code[pos*2+2:pos*2+6], 16, 64)
				pos += (int64(opCode.Immediates) + 1 + int64(offset))
			case opCode.Name == "RJUMPI":
				if int(pos)*2+6 > len(code) {
					fmt.Println("Error: Truncted RJUMPI")
					break
				}

				offset, _ := strconv.ParseInt(code[pos*2+2:pos*2+6], 16, 64)

				if offset > 32767 {
					offset = ((65535 - offset) + 1) * -1
				}

				worklist = append(worklist, []int64{pos + 3 + offset, stackHeight})
				fmt.Println(code[pos*2+2 : pos*2+6])
				pos += int64(opCode.Immediates) + 1
			case opCode.Name == "RJUMPV":
				if int(pos)*2+4 > len(code) {
					fmt.Println("Error: truncated RJUMPV")
					break
				}
				count, _ := strconv.ParseInt(code[pos*2+2:pos*2+4], 16, 64)
				fmt.Println("\tcount:", count)

				pos += 2
				fmt.Println(pos*2+count*2, len(code))
				if int(pos)*2+int(count)*2 > len(code) {
					fmt.Println("Error: truncated RJUMPV.")
					break
				}
				for i := 0; i < int(count); i++ {
					offset, _ := strconv.ParseInt(code[int(pos)*2+4*i:int(pos)*2+4*i+4], 16, 64)

					if offset > 32767 {
						offset = ((65535 - offset) + 1) * -1
					}

					fmt.Println("\toffset:", offset)
					fmt.Println("\twE:", pos+2*count+offset)
					worklist = append(worklist, []int64{pos + 2*count + offset, stackHeight})
				}
				fmt.Println("\tcode:", code[pos*2:])
				pos += 2 * count
				fmt.Println("\tcode:", code[pos*2:])
			default:
				if opCode.IsTerminating {
					break outer
				} else {
					pos += int64(opCode.Immediates) + 1
				}
			}

			maxStackHeight = int64(math.Max(float64(maxStackHeight), float64(stackHeight)))
		}

		if maxStackHeight > 1024 {
			fmt.Println("Error: max stack above limit")
		}

		if len(worklist) == 0 {
			break
		}
	}
	fmt.Println(">> heights:", len(stackHeights))
	return maxStackHeight
}

func ParseEOF(eof_code string) (EOFObject, error) {
	version := int64(0)
	versionHex := ""
	eof_code = strings.ToLower(eof_code)

	codeHeaders := []int64{}
	typesLength := int64(0)
	types := [][]int64{}
	dataLength := int64(0)
	dataContent := ""

	result := NewEOFObject()

	i := 0
	for {
		if i+2 > len(eof_code) {
			break
		}

		b := eof_code[i : i+2]

		if versionHex == "" && b != "ef" {
			return result, errors.New("Invalid EOF code")
		}

		if versionHex == "" && b == "ef" {
			versionHex = eof_code[i+2 : i+6]
			err := errors.New("")
			version, err = strconv.ParseInt(versionHex, 16, 64)

			if err != nil {
				return result, errors.New("Invalid version")
			}
			result.Version = version
			i += 4
		}

		if b == "01" {
			typesLengthHex := eof_code[i+2 : i+6]
			err := errors.New("")
			typesLength, err = strconv.ParseInt(typesLengthHex, 16, 64)

			if err != nil {
				return result, errors.New("Invalid types length")
			}
			i += 4
		}

		if b == "02" {
			codeSectionsTotalHex := eof_code[i+2 : i+6]
			codeSectionsTotal, err := strconv.ParseInt(codeSectionsTotalHex, 16, 64)

			if err != nil {
				return result, errors.New("Invalid code sections total")
			}

			i += 6
			for j := 0; j < int(codeSectionsTotal); j++ {
				codeLenHex := eof_code[i : i+4]
				codeLen, err := strconv.ParseInt(codeLenHex, 16, 64)

				if err != nil {
					return result, errors.New("Invalid code section length")
				}

				codeHeaders = append(codeHeaders, codeLen)
				i += 4
			}
			i -= 2
		}

		if b == "03" {
			err := errors.New("")
			dataLengthHex := eof_code[i+2 : i+6]
			dataLength, err = strconv.ParseInt(dataLengthHex, 16, 64)

			if err != nil {
				return result, errors.New("Invalid data section length")
			}

			i += 4
		}

		if b == "00" {
			for j := 0; j < int(typesLength); j += 4 {
				inputsHex := eof_code[i+2 : i+4]
				inputs, err := strconv.ParseInt(inputsHex, 16, 64)

				if err != nil {
					return result, errors.New("Invalid type input")
				}

				outputsHex := eof_code[i+4 : i+6]
				outputs, err := strconv.ParseInt(outputsHex, 16, 64)

				if err != nil {
					return result, errors.New("Invalid type output")
				}

				maxStackHex := eof_code[i+6 : i+10]
				maxStack, err := strconv.ParseInt(maxStackHex, 16, 64)

				if err != nil {
					return result, errors.New("Invalid Max Stack Height")
				}

				types = append(types, []int64{inputs, outputs, maxStack})
				i += 8
			}
			i += 2

			// Extract code
			for j, ch := range codeHeaders {
				code := eof_code[i : i+int(ch)*2]
				result.AddCodeWithType(code, types[j])
				i += int(ch) * 2
			}

			// Extract data
			dataContent = eof_code[i : i+int(dataLength)*2]
			result.Data = dataContent
			i += int(dataLength) * 2
		}
		i += 2
	}

	return result, nil
}

func ParseOldEOF(eof_code string) EOFObject {
	version := int64(0)
	versionHex := ""
	eof_code = strings.ToLower(eof_code)

	codeHeaders := []int64{}
	codeSections := []string{}
	typesLength := int64(0)
	types := [][]int64{}
	dataLength := int64(0)
	dataContent := ""

	i := 0
	for {
		if i+2 > len(eof_code) {
			break
		}

		bt := eof_code[i : i+2]
		fmt.Println("bt: ", bt)

		if versionHex == "" && bt != "ef" {
			fmt.Println("Error: Invalid EOF code")
			break
		}

		if versionHex == "" && bt == "ef" {
			versionHex = eof_code[i+2 : i+6]
			version, _ = strconv.ParseInt(versionHex, 16, 64)
			i += 4
		}

		if bt == "03" {
			fmt.Println(">types")
			fmt.Println("code: ", eof_code[i:])
			typesLengthHex := eof_code[i+2 : i+6]
			typesLengthTmp, err := strconv.ParseInt(typesLengthHex, 16, 64)

			if err != nil {
				fmt.Println("Error: Invalid types legnth : ", err)
			}

			typesLength = typesLengthTmp
			i += 4
		}

		if bt == "01" {
			fmt.Println(">code")
			fmt.Println("code: ", eof_code[i:])
			codeLenHex := eof_code[i+2 : i+6]
			codeLen, err := strconv.ParseInt(codeLenHex, 16, 64)

			if err != nil {
				fmt.Println("Error: Invalid code length :", err)
			}

			codeHeaders = append(codeHeaders, codeLen)
			i += 4
		}

		if bt == "02" {
			fmt.Println(">data")
			fmt.Println("code: ", eof_code[i:])
			dataLengthHex := eof_code[i+2 : i+6]
			dataLength, _ = strconv.ParseInt(dataLengthHex, 16, 64)
			i += 4
		}

		if bt == "00" { // Terminator
			fmt.Println("> terminator")
			fmt.Println("code: ", eof_code[i:])
			// Extract Types
			if typesLength > 0 {
				for j := 0; j < int(typesLength); j += 2 {
					inputsHex := eof_code[i+2 : i+4]
					inputs, _ := strconv.ParseInt(inputsHex, 16, 64)
					outputsHex := eof_code[i+4 : i+6]
					outputs, _ := strconv.ParseInt(outputsHex, 16, 64)

					types = append(types, []int64{inputs, outputs})
					i += 4
				}
			}
			fmt.Println("types: ", types)

			i += 2
			fmt.Println("code: ", eof_code[i:])
			fmt.Println("codeHeaders: ", codeHeaders)
			// Extract Code
			for _, cH := range codeHeaders {
				code := eof_code[i : i+int(cH)*2]
				fmt.Println("codeX: ", code)
				codeSections = append(codeSections, code)
				i += int(cH) * 2
			}

			// Extract Data
			dataContent = eof_code[i : i+int(dataLength)*2]
			i += int(dataLength) * 2
		}
		i += 2
	}
	return EOFObject{
		Version:      version,
		CodeSections: codeSections,
		Types:        types,
		Data:         dataContent,
	}
}
