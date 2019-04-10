package linttest_test

import (
	"log"
	"testing"

	"github.com/VKCOM/noverify/src/linter"
	"github.com/VKCOM/noverify/src/linttest"
	"github.com/VKCOM/noverify/src/meta"
)

func TestCallStatic(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	class T {
		public static function sf($_) {}
		public function f($_) {}
	}
	$v = new T();
	$v->sf(1);
	T::f(1);
	`)
	test.Expect = []string{
		`Calling static method as instance method`,
		`Calling instance method as static method`,
	}
	runFilterMatch(test, "callStatic")
}

func TestSwitchContinue1(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	global $x;
	global $y;

	switch ($x) {
	case 10:
		continue;
	}

	switch ($x) {
	case 10:
		if ($x == $y) {
			continue;
		}
	}

	for ($i = 0; $i < 10; $i++) {
		switch ($i) {
		case 5:
			continue;
		}
	}`)
	test.Expect = []string{
		`'continue' inside switch is 'break'`,
		`'continue' inside switch is 'break'`,
		`'continue' inside switch is 'break'`,
	}
	test.RunAndMatch()
}

func TestSwitchContinue2(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	global $x;
	switch ($x) {
	case 10:
		for ($i = 0; $i < 10; $i++) {
			if ($i == $x) {
				continue; // OK, bound to 'for'
			}
		}
	}

	// OK, "continue 2" does the right thing.
	// Phpstorm finds incorrect label "level" values,
	// but it doesn't report 'continue' (without level) as being bad.
	for ($i = 0; $i < 3; $i++) {
		switch ($x) {
		case 10:
			continue 2;
		}
	}`)
	test.RunAndMatch()
}

func TestBuiltinConstant(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function f() {
		$_ = NULL;
		$_ = True;
		$_ = FaLsE;
	}`)
	test.Expect = []string{
		"Use null instead of NULL",
		"Use true instead of True",
		"Use false instead of FaLsE",
	}
	test.RunAndMatch()
}

func TestFunctionNotOnlyExits2(t *testing.T) {
	linttest.SimpleNegativeTest(t, `<?php
	function rand() {
		return 4;
	}

	class RuntimeException {}

	class Something {
		/** may throw */
		public static function doExit() {
			if (rand()) {
				throw new \RuntimeException("OMG");
			}

			return rand();
		}
	}

	function doSomething() {
		Something::doExit();
		echo "Not always dead code";
	}`)
}

func TestArrayAccessForClass(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	class three {}
	class five {}
	function test() {
		$a = 1==2 ? new three : new five;
		return $a['test'];
	}`)
	test.Expect = []string{"Array access to non-array type"}
	test.RunAndMatch()
}

// This test checks that expressions are evaluated in correct order.
// If order is incorrect then there would be an error that we are referencing elements of a class
// that does not implement ArrayAccess.
func TestCorrectTypes(t *testing.T) {
	linttest.SimpleNegativeTest(t, `<?php
	class three {}
	class five {}
	function test() {
		$a = ['test' => 1];
		$a = ($a['test']) ? new three : new five;
		return $a;
	}`)
}

func TestAllowReturnAfterUnreachable(t *testing.T) {
	linttest.SimpleNegativeTest(t, `<?php
	function unreachable() {
		exit;
	}

	function test() {
		unreachable();
		return;
	}`)
}

func TestFunctionReferenceParams(t *testing.T) {
	linttest.SimpleNegativeTest(t, `<?php
	function doSometing(&$result) {
		$result = 5;
	}`)
}

func TestFunctionReferenceParamsInAnonymousFunction(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function doSometing() {
		return function() use($a, &$result) {
			echo $a;
			$result = 1;
		};
	}`)
	test.Expect = []string{"Undefined variable a"}
	test.RunAndMatch()
}

func TestForeachByRefUnused(t *testing.T) {
	linttest.SimpleNegativeTest(t, `<?php
	class SomeClass {
		public $a;
	}

	/**
	 * @param SomeClass[] $some_arr
	 */
	function doSometing() {
		$some_arr = [];

		foreach ($some_arr as $var) {
			$var->a = 1;
		}

		foreach ($some_arr as &$var2) {
			$var2->a = 2;
		}
	}`)
}

func TestAllowAssignmentInForLoop(t *testing.T) {
	linttest.SimpleNegativeTest(t, `<?php
	function test() {
	  for ($day = 0; $day <= 100; $day = $day + 1) {
		echo $day;
	  }
	}
	`)
}

func TestDuplicateArrayKey(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function test() {
	  return [
		  'key1' => 'something',
		  'key2' => 'other_thing',
		  'key1' => 'third_thing', // duplicate
	  ];
	}`)
	test.Expect = []string{"Duplicate array key 'key1'"}
	test.RunAndMatch()
}

func TestMixedArrayKeys(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function test() {
	  return [
		  'something',
		  'key2' => 'other_thing',
		  'key3' => 'third_thing',
	  ];
	}
	`)
	test.Expect = []string{"Mixing implicit and explicit array keys"}
	test.RunAndMatch()
}

func TestStringGlobalVarName(t *testing.T) {
	// Should not panic.
	linttest.SimpleNegativeTest(t, `<?php
	function f() {
		global ${"x"};
		global ${"${x}_{$x}"};
	}`)
}

func TestArrayLiteral(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function traditional_array_literal() {
		return array(1, 2);
	}`)
	test.Expect = []string{"Use of old array syntax"}
	test.RunAndMatch()
}

func TestIssetVarVar4(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function issetVarVar() {
		if (isset($$$$x)) {
			$_ = $$$$x; // Can't track this level of indirection
		}
	}`)
	test.Expect = []string{
		"Unknown variable variable $$$x used",
		"Unknown variable variable $$$$x used",
	}
	test.RunAndMatch()
}

func TestIssetVarVar3(t *testing.T) {
	// Test that irrelevant isset of variable-variable doesn't affect
	// other variables. Also warn for undefined variable in $$x.
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function issetVarVar() {
		if (isset($$x)) {
			$_ = $$y;
		}
	}`)
	test.Expect = []string{
		"Undefined variable: x",
		"Unknown variable variable $$y used",
	}
	test.RunAndMatch()
}

func TestIssetVarVar2(t *testing.T) {
	// Test that if $x is defined, it doesn't make $$x defined.
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function issetVarVar() {
		if (isset($x)) {
			$_ = $x;  // $x is defined
			$_ = $$x; // But $$x is not
		}
	}`)
	test.Expect = []string{"Unknown variable variable $$x used"}
	test.RunAndMatch()
}

func TestIssetVarVar1(t *testing.T) {
	// Test that defined variable variable don't cause "undefined" warnings.
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function issetVarVar() {
		$x = 'key';
		if (isset($$x)) {
			$_ = $x + 1;  // If $$x is isset, then $x is set as well
			$_ = $$x + 1;
			$_ = $y;      // Undefined
		}
		// After the block all vars are undefined again.
		$_ = $x;
	}`)
	test.Expect = []string{"Undefined variable: y"}
	test.RunAndMatch()
}

func TestUnused(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function unused_test($arg1, $arg2) {
		global $g;

		$_SERVER['test'] = 1; // superglobal, must not count as unused

		$_ = 'should not count as unused';
		$a = 10;
		foreach ([1, 2, 3] as $k => $v) {
			// $v is unused here
			echo $k;
		}
	}`)
	test.Expect = []string{
		"Unused variable g ",
		"Unused variable a ",
		"Unused variable v ",
	}
	test.RunAndMatch()
}

func TestAtVar(t *testing.T) {
	// variables declared using @var should not be overriden
	_ = linttest.GetFileReports(t, `<?php
	function test() {
		/** @var string $a */
		$a = true;
		return $a;
	}`)

	fi, ok := meta.Info.GetFunction(`\test`)
	if !ok {
		t.Errorf("Could not get function test")
	}

	typ := fi.Typ
	hasBool := false
	hasString := false

	typ.Iterate(func(typ string) {
		if typ == "string" {
			hasString = true
		} else if typ == "bool" {
			hasBool = true
		}
	})

	log.Printf("$a type = %s", typ)

	if !hasBool {
		t.Errorf("Type of variable a does not have boolean type")
	}

	if !hasString {
		t.Errorf("Type of variable a does not have string type")
	}
}

func TestFunctionExit(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php function doExit() {
		exit;
	}

	function doSomething() {
		doExit();
		echo "Dead code";
	}`)
	test.Expect = []string{"Unreachable code"}
	test.RunAndMatch()
}

func TestFunctionNotOnlyExits(t *testing.T) {
	linttest.SimpleNegativeTest(t, `<?php function rand() {
		return 4;
	}

	function doExit() {
		if (rand()) {
			exit;
		} else {
			return;
		}
	}

	function doSomething() {
		doExit();
		echo "Not always dead code";
	}`)
}

func TestFunctionJustReturns(t *testing.T) {
	linttest.SimpleNegativeTest(t, `<?php function justReturn() {
		return 1;
	}

	function doSomething() {
		justReturn();
		echo "Just normal code";
	}`)
}

func TestSwitchFallthrough(t *testing.T) {
	linttest.SimpleNegativeTest(t, `<?php
	function withFallthrough($a) {
		switch ($a) {
		case 1:
			echo "1\n";
			// With prepended comment line.
			// fallthrough
		case 2:
			echo "2\n";
			// falls through and continue rolling
		case 3:
			echo "3\n";
			/* fallthrough and blah-blah */
		case 4:
			echo "4\n";
			/* falls through */
		default:
			echo "Other\n";
		}
	}`)
}

func TestFunctionThrowsExceptionsAndReturns(t *testing.T) {
	reports := linttest.GetFileReports(t, `<?php
	class Exception {}

	function handle($b) {
		if ($b === 1) {
			return $b;
		}

		switch ($b) {
			case "a":
				throw new \Exception("a");

			default:
				throw new \Exception("default");
		}
	}

	function doSomething() {
		handle(1);
		echo "This code is reachable\n";
	}`)

	if len(reports) != 0 {
		t.Errorf("Unexpected number of reports: expected 0, got %d", len(reports))
	}

	fi, ok := meta.Info.GetFunction(`\handle`)

	if ok {
		log.Printf("handle exitFlags: %d (%s)", fi.ExitFlags, linter.FlagsToString(fi.ExitFlags))
	}

	for _, r := range reports {
		log.Printf("%s", r)
	}
}

func TestRedundantCast(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function bad($a) {
		$int = 1;
		$double = 1.0;
		$string = '1';
		$bool = ($a == 0);
		$array = [1, 'a', 3.0]; // Mixed elems on purpose
		$a = (int)$int;
		$a = (double)$double;
		$a = (string)$string;
		$a = (bool)$bool;
		$a = (array)$array;
		$_ = $a;
	}

	function good($a) {
		$int = 1;
		$double = 1.0;
		$string = '1';
		$bool = ($a == 0);
		$array = [1, 'a', 3.0];
		$a = (int)$double;
		$a = (double)$array;
		$a = (string)$bool;
		$a = (bool)$string;
		$a = (array)$int;
		$_ = $a;
	}`)
	test.Expect = []string{
		`expression already has array type`,
		`expression already has double type`,
		`expression already has int type`,
		`expression already has string type`,
		`expression already has bool type`,
	}
	test.RunAndMatch()
}

func TestSwitchBreak(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function bad($a) {
		switch ($a) {
		case 1:
			echo "One\n"; // Bad, no break.
		default:
			echo "Other\n";
		}
	}

	function good($a) {
		switch ($a) {
		case 1:
			echo "One\n";
			break;
		case 2:
			echo "Two";
			// No break, but still good, since it's the last case clause.
		}

		echo "Three";
	}`)
	test.Expect = []string{`Add break or '// fallthrough' to the end of the case`}
	test.RunAndMatch()
}

func TestCorrectArrayTypes(t *testing.T) {
	test := linttest.NewSuite(t)
	test.AddFile(`<?php
	function test() {
		$a = [ 'a' => 123, 'b' => 3456 ];
		return $a['a'];
	}
	`)
	test.RunLinter()

	fn, ok := meta.Info.GetFunction(`\test`)
	if !ok {
		t.Errorf("Could not find function test")
		t.Fail()
	}

	if l := fn.Typ.Len(); l != 1 {
		t.Errorf("Unexpected number of types: %d, excepted 1", l)
	}

	if !fn.Typ.IsInt() {
		t.Errorf("Wrong type: %s, excepted int", fn.Typ)
	}
}

func runFilterMatch(test *linttest.Suite, name string) {
	test.Match(filterReports(name, test.RunLinter()))
}

func filterReports(name string, reports []*linter.Report) []*linter.Report {
	var out []*linter.Report
	for _, r := range reports {
		if r.CheckName() == name {
			out = append(out, r)
		}
	}
	return out
}
