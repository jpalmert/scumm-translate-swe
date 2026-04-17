"""Tests for find_dynamic_names.py — setObjectName parsing from descumm output."""

import sys
import os
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from find_dynamic_names import find_setobjectname_targets


class TestFindSetObjectNameTargets(unittest.TestCase):
    """Test parsing of descumm decompilation output."""

    def test_simple_setobjectname(self):
        text = '''\
[0004] (6E)   setObjectName(95, "a mug of grog");
'''
        targets = find_setobjectname_targets(text)
        self.assertEqual(len(targets), 1)
        obj_id, string_idx = targets[0]
        self.assertEqual(obj_id, 95)
        self.assertEqual(string_idx, 0)

    def test_multiple_setobjectname(self):
        text = '''\
[0001] (6E)   setObjectName(95, "mug");
[0010] (6E)   setObjectName(96, "grog");
'''
        targets = find_setobjectname_targets(text)
        self.assertEqual(len(targets), 2)
        self.assertEqual(targets[0], (95, 0))
        self.assertEqual(targets[1], (96, 1))

    def test_var_me_with_verb_obj_id(self):
        text = '''\
[0004] (6E)   setObjectName(VAR_ME, "closed door");
'''
        targets = find_setobjectname_targets(text, verb_obj_id=42)
        self.assertEqual(len(targets), 1)
        self.assertEqual(targets[0], (42, 0))

    def test_var_me_without_verb_obj_id(self):
        text = '''\
[0004] (6E)   setObjectName(VAR_ME, "closed door");
'''
        targets = find_setobjectname_targets(text, verb_obj_id=None)
        # VAR_ME without verb_obj_id cannot be resolved
        self.assertEqual(len(targets), 0)

    def test_local_variable_skipped(self):
        text = '''\
[0004] (6E)   setObjectName(Local[3], "something");
'''
        targets = find_setobjectname_targets(text)
        # Local variable targets cannot be resolved
        self.assertEqual(len(targets), 0)

    def test_no_setobjectname(self):
        text = '''\
[0000] (40)   cutscene() {
[0004] (80)     actorOps.setCurActor(4);
[0008] (A8)   stopScript(0);
'''
        targets = find_setobjectname_targets(text)
        self.assertEqual(len(targets), 0)

    def test_string_counting_with_other_strings(self):
        # Other quoted strings before setObjectName increment the string index
        text = '''\
[0000] (26)   print(255, "Hello world");
[0010] (26)   print(255, "Goodbye");
[0020] (6E)   setObjectName(100, "renamed thing");
'''
        targets = find_setobjectname_targets(text)
        self.assertEqual(len(targets), 1)
        obj_id, string_idx = targets[0]
        self.assertEqual(obj_id, 100)
        # Three lines with quotes: index 0, 1, 2
        self.assertEqual(string_idx, 2)

    def test_empty_input(self):
        targets = find_setobjectname_targets("")
        self.assertEqual(len(targets), 0)

    def test_setobjectname_string_index_starts_at_negative_one(self):
        # Lines without quotes don't increment the counter
        text = '''\
[0000] (40)   cutscene() {
[0004] (6E)   setObjectName(50, "new name");
'''
        targets = find_setobjectname_targets(text)
        self.assertEqual(len(targets), 1)
        # Only one line with quotes (the setObjectName line itself), index = 0
        self.assertEqual(targets[0], (50, 0))


if __name__ == "__main__":
    unittest.main()
