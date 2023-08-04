// Copyright 2023 The concrete-geth Authors
//
// The concrete-geth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The concrete library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the concrete library. If not, see <http://www.gnu.org/licenses/>.

package testing

import (
	_ "embed"
	"testing"
)

//go:embed testdata/out/Test.sol/Test.json
var testContractJsonBytes []byte

func TestRunTestContract(t *testing.T) {
	bytecode, ABI, _, err := extractTestData(testContractJsonBytes)
	if err != nil {
		t.Fatal(err)
	}
	passed, failed := runTestContract(bytecode, ABI)
	if failed > 0 {
		t.Errorf("failed tests: %v", failed)
	}
	if passed == 0 {
		t.Error("no tests passed")
	}
}

// func TestRunTests(t *testing.T) {
// 	testPaths, err := getTestPaths("./testdata/src", "testdata/out")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if len(testPaths) == 0 {
// 		t.Fatal("no test paths found")
// 	}
// 	runTestPaths(testPaths)
// }
