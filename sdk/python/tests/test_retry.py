#!/usr/bin/env python

import itertools
import unittest

import arvados.errors as arv_error
import arvados.retry as arv_retry
import mock

from arvados_testutil import fake_httplib2_response

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


@mock.patch('time.sleep')
class HTTPRetryLoopTestCase(unittest.TestCase):
    def run_loop(self, num_retries, *codes, **kwargs):
        responses = itertools.chain(
            ((fake_httplib2_response(c), str(c)) for c in codes),
            itertools.repeat((None, None)))
        retrier = arv_retry.HTTPRetryLoop(num_retries, **kwargs)
        for tries_left, response in itertools.izip(retrier, responses):
            retrier.save_result(response)
        return retrier

    def check_result(self, retrier, expect_success, last_status,
                     sleep_mock, sleep_count):
        self.assertIs(retrier.success(), expect_success,
                      "loop success flag is incorrect")
        self.assertEqual(str(last_status), retrier.last_result()[1],
                         "wrong loop result")
        self.assertEqual(sleep_count, sleep_mock.call_count,
                         "loop did not back off correctly")

    def sleep_times(self, sleep_mock):
        return (arglist[0][0] for arglist in sleep_mock.call_args_list)

    def check_backoff_growth(self, sleep_mock, multiplier=1):
        check = (self.assertGreater if (multiplier == 1)
                 else self.assertGreaterEqual)
        sleep_times = self.sleep_times(sleep_mock)
        last_wait = next(sleep_times)
        for this_wait in sleep_times:
            check(this_wait, last_wait * multiplier,
                  "loop did not grow backoff times correctly")
            last_wait = this_wait

    def test_no_retries_and_success(self, sleep_mock):
        retrier = self.run_loop(0, 200)
        self.check_result(retrier, True, 200, sleep_mock, 0)

    def test_no_retries_and_tempfail(self, sleep_mock):
        retrier = self.run_loop(0, 500, 200)
        self.check_result(retrier, None, 500, sleep_mock, 0)

    def test_no_retries_and_permfail(self, sleep_mock):
        retrier = self.run_loop(0, 400, 200)
        self.check_result(retrier, False, 400, sleep_mock, 0)

    def test_retries_with_immediate_success(self, sleep_mock):
        retrier = self.run_loop(3, 200, 500, 500)
        self.check_result(retrier, True, 200, sleep_mock, 0)

    def test_retries_with_delayed_success(self, sleep_mock):
        retrier = self.run_loop(3, 500, 500, 200, 502)
        self.check_result(retrier, True, 200, sleep_mock, 2)
        self.check_backoff_growth(sleep_mock)

    def test_retries_then_permfail(self, sleep_mock):
        retrier = self.run_loop(3, 500, 404, 200, 200)
        self.check_result(retrier, False, 404, sleep_mock, 1)

    def test_retries_all_tempfail(self, sleep_mock):
        retrier = self.run_loop(3, 502, 502, 502, 500, 200)
        self.check_result(retrier, None, 500, sleep_mock, 3)
        self.check_backoff_growth(sleep_mock)

    def test_backoff_parameters(self, sleep_mock):
        with mock.patch('time.time', side_effects=itertools.count):
            self.run_loop(3, 500, 500, 500, 500,
                          backoff_start=5, backoff_growth=10)
        self.check_backoff_growth(sleep_mock, 10)

    def test_custom_success_check(self, mock):
        retrier = self.run_loop(3, 200, 777, 201, 202, 203,
                                success_check=lambda r: r[1] == '777' or None)
        self.check_result(retrier, True, 777, mock, 1)


class CheckHTTPResponseSuccessTestCase(unittest.TestCase):
    def results_map(self, *codes):
        for code in codes:
            response = (fake_httplib2_response(code), None)
            yield code, arv_retry.check_http_response_success(response)

    def check(assert_name):
        def check_method(self, expected, *codes):
            assert_func = getattr(self, assert_name)
            for code, actual in self.results_map(*codes):
                assert_func(expected, actual,
                            "{} status flagged {}".format(code, actual))
                if assert_name != 'assertIs':
                    self.assertTrue(
                        actual is True or actual is False or actual is None,
                        "{} status returned {}".format(code, actual))
        return check_method

    check_is = check('assertIs')
    check_is_not = check('assertIsNot')

    def test_obvious_successes(self):
        self.check_is(True, *range(200, 207))

    def test_obvious_stops(self):
        self.check_is(False, 424, 426, 428, 431,
                      *range(400, 408) + range(410, 420))

    def test_obvious_retries(self):
        self.check_is(None, 500, 502, 503, 504)

    def test_4xx_retries(self):
        self.check_is(None, 408, 409, 422, 423)

    def test_5xx_failures(self):
        self.check_is(False, 501, *range(505, 512))

    def test_1xx_not_retried(self):
        self.check_is_not(None, 100, 101)

    def test_redirects_not_retried(self):
        self.check_is_not(None, *range(300, 309))

    def test_wacky_code_retries(self):
        self.check_is(None, 0, 99, 600, -200)


if __name__ == '__main__':
    unittest.main()
