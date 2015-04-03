import arvados
import unittest
import arvados_testutil as tutil
import arvados.commands.cwl_job as cwl_job

class CwlJobTestCase(unittest.TestCase):
    def test_parse_sinfo(self):
        nodes = cwl_job.parse_sinfo("""
16 compute0,compute2
8 compute1,compute[3-5],compute7
""")
        self.assertEqual({"compute0": {"slots": 16},
                          "compute2": {"slots": 16},
                          "compute1": {"slots": 8},
                          "compute3": {"slots": 8},
                          "compute4": {"slots": 8},
                          "compute5": {"slots": 8},
                          "compute7": {"slots": 8}},
        nodes)


    def test_make_slots(self):
        slots = cwl_job.make_slots({"compute0": {"slots": 2}, "compute1": {"slots": 4}})
        self.assertEqual({"compute0[0]": {"node": "compute0", "slot": 0, "task": None},
                          "compute0[1]": {"node": "compute0", "slot": 1, "task": None},
                          "compute1[0]": {"node": "compute1", "slot": 0, "task": None},
                          "compute1[1]": {"node": "compute1", "slot": 1, "task": None},
                          "compute1[2]": {"node": "compute1", "slot": 2, "task": None},
                          "compute1[3]": {"node": "compute1", "slot": 3, "task": None}},
                          slots)
