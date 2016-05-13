// Copyright 2015, 2016 Eris Industries (UK) Ltd.
// This file is part of Eris-RT

// Eris-RT is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Eris-RT is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with Eris-RT.  If not, see <http://www.gnu.org/licenses/>.

package definitions

type Do struct {
	Debug        bool
	Verbose      bool
	ChainId      string
	// ChainType    string
	// CSV          string
	// AccountTypes []string
	// Zip          bool
	// Tarball      bool
	Output       bool
	// Accounts     []*Account
	// Result       string
}

func NowDo() *Do {
	return &Do{}
}
