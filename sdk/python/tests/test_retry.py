#!/usr/bin/env python

import itertools
import unittest

import arvados.errors as arv_error
import arvados.retry as arv_retry

class RetryLoopTestCase(unittest.TestCase):
    @staticmethod
    def loop_success(result):
        # During the tests, we use integers that look like HTTP status
        # codes as loop results.  Then we define simplified HTTP
        # heuristics here to decide whether the result is success (True),
        # permanent failure (False), or temporary failure (None).
        if result < 400:
            return True
        elif result < 500:
            return False
        else:
            return None

    def run_loop(self, num_retries, *results):
        responses = itertools.chain(results, itertools.repeat(None))
        retrier = arv_retry.RetryLoop(num_retries, self.loop_success)
        for tries_left, response in itertools.izip(retrier, responses):
            retrier.save_result(response)
        return retrier

    def check_result(self, retrier, expect_success, last_code):
        self.assertIs(retrier.success(), expect_success,
                      "loop success flag is incorrect")
        self.assertEqual(last_code, retrier.last_result())

    def test_zero_retries_and_success(self):
        retrier = self.run_loop(0, 200)
        self.check_result(retrier, True, 200)

    def test_zero_retries_and_tempfail(self):
        retrier = self.run_loop(0, 500, 501)
        self.check_result(retrier, None, 500)

    def test_zero_retries_and_permfail(self):
        retrier = self.run_loop(0, 400, 201)
        self.check_result(retrier, False, 400)

    def test_one_retry_with_immediate_success(self):
        retrier = self.run_loop(1, 200, 201)
        self.check_result(retrier, True, 200)

    def test_one_retry_with_delayed_success(self):
        retrier = self.run_loop(1, 500, 201)
        self.check_result(retrier, True, 201)

    def test_one_retry_with_no_success(self):
        retrier = self.run_loop(1, 500, 501, 502)
        self.check_result(retrier, None, 501)

    def test_one_retry_but_permfail(self):
        retrier = self.run_loop(1, 400, 201)
        self.check_result(retrier, False, 400)

    def test_two_retries_with_immediate_success(self):
        retrier = self.run_loop(2, 200, 201, 202)
        self.check_result(retrier, True, 200)

    def test_two_retries_with_success_after_one(self):
        retrier = self.run_loop(2, 500, 201, 502)
        self.check_result(retrier, True, 201)

    def test_two_retries_with_success_after_two(self):
        retrier = self.run_loop(2, 500, 501, 202, 503)
        self.check_result(retrier, True, 202)

    def test_two_retries_with_no_success(self):
        retrier = self.run_loop(2, 500, 501, 502, 503)
        self.check_result(retrier, None, 502)

    def test_two_retries_with_permfail(self):
        retrier = self.run_loop(2, 500, 401, 202)
        self.check_result(retrier, False, 401)

    def test_save_result_before_start_is_error(self):
        retrier = arv_retry.RetryLoop(0)
        self.assertRaises(arv_error.AssertionError, retrier.save_result, 1)

    def test_save_result_after_end_is_error(self):
        retrier = arv_retry.RetryLoop(0)
        for count in retrier:
            pass
        self.assertRaises(arv_error.AssertionError, retrier.save_result, 1)


if __name__ == '__main__':
    unittest.main()
