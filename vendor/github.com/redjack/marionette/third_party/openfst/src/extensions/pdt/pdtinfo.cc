// pdtinfo.cc

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright 2005-2010 Google, Inc.
// Author: riley@google.com (Michael Riley)
//
// \file
// Prints out various information about a PDT such as number of
// states, arcs and parentheses.
//

#include <fst/extensions/pdt/pdtscript.h>
#include <fst/util.h>

DEFINE_string(pdt_parentheses, "", "PDT parenthesis label pairs.");

int main(int argc, char **argv) {
  namespace s = fst::script;

  string usage = "Prints out information about a PDT.\n\n  Usage: ";
  usage += argv[0];
  usage += " in.pdt\n";


  std::set_new_handler(FailedNewHandler);
  SET_FLAGS(usage.c_str(), &argc, &argv, true);
  if (argc > 2) {
    ShowUsage();
    return 1;
  }

  string in_name = (argc > 1 && (strcmp(argv[1], "-") != 0)) ? argv[1] : "";

  s::FstClass *ifst = s::FstClass::Read(in_name);
  if (!ifst) return 1;

  if (FLAGS_pdt_parentheses.empty()) {
    LOG(ERROR) << argv[0] << ": No PDT parenthesis label pairs provided";
    return 1;
  }

  vector<pair<int64, int64> > parens;
  fst::ReadLabelPairs(FLAGS_pdt_parentheses, &parens, false);

  s::PrintPdtInfo(*ifst, parens);

  return 0;
}
