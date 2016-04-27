#!/usr/bin/env python

import itertools
import unittest

import arvados.errors as arv_error
import arvados.retry as arv_retry
import mock

class RetryLoopTestMixin(object):
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

    def run_loop(self, num_retries, *results, **kwargs):
        responses = itertools.chain(results, itertools.repeat(None))
        retrier = arv_retry.RetryLoop(num_retries, self.loop_success,
                                      **kwargs)
        for tries_left, response in itertools.izip(retrier, responses):
            retrier.save_result(response)
        return retrier

    def check_result(self, retrier, expect_success, last_code):
        self.assertIs(retrier.success(), expect_success,
                      "loop success flag is incorrect")
        self.assertEqual(last_code, retrier.last_result())


class RetryLoopTestCase(unittest.TestCase, RetryLoopTestMixin):
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


@mock.patch('time.time', side_effect=itertools.count())
@mock.patch('time.sleep')
class RetryLoopBackoffTestCase(unittest.TestCase, RetryLoopTestMixin):
    def run_loop(self, num_retries, *results, **kwargs):
        kwargs.setdefault('backoff_start', 8)
        return super(RetryLoopBackoffTestCase, self).run_loop(
            num_retries, *results, **kwargs)

    def check_backoff(self, sleep_mock, sleep_count, multiplier=1):
        # Figure out how much time we actually spent sleeping.
        sleep_times = [arglist[0][0] for arglist in sleep_mock.call_args_list
                       if arglist[0][0] > 0]
        self.assertEqual(sleep_count, len(sleep_times),
                         "loop did not back off correctly")
        last_wait = 0
        for this_wait in sleep_times:
            self.assertGreater(this_wait, last_wait * multiplier,
                               "loop did not grow backoff times correctly")
            last_wait = this_wait

    def test_no_backoff_with_no_retries(self, sleep_mock, time_mock):
        self.run_loop(0, 500, 201)
        self.check_backoff(sleep_mock, 0)

    def test_no_backoff_after_success(self, sleep_mock, time_mock):
        self.run_loop(1, 200, 501)
        self.check_backoff(sleep_mock, 0)

    def test_no_backoff_after_permfail(self, sleep_mock, time_mock):
        self.run_loop(1, 400, 201)
        self.check_backoff(sleep_mock, 0)

    def test_backoff_before_success(self, sleep_mock, time_mock):
        self.run_loop(5, 500, 501, 502, 203, 504)
        self.check_backoff(sleep_mock, 3)

    def test_backoff_before_permfail(self, sleep_mock, time_mock):
        self.run_loop(5, 500, 501, 502, 403, 504)
        self.check_backoff(sleep_mock, 3)

    def test_backoff_all_tempfail(self, sleep_mock, time_mock):
        self.run_loop(3, 500, 501, 502, 503, 504)
        self.check_backoff(sleep_mock, 3)

    def test_backoff_multiplier(self, sleep_mock, time_mock):
        self.run_loop(5, 500, 501, 502, 503, 504, 505,
                      backoff_start=5, backoff_growth=10, max_wait=1000000000)
        self.check_backoff(sleep_mock, 5, 9)


class CheckHTTPResponseSuccessTestCase(unittest.TestCase):
    def results_map(self, *codes):
        for code in codes:
            yield code, arv_retry.check_http_response_success(code)

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


class RetryMethodTestCase(unittest.TestCase):
    class Tester(object):
        def __init__(self):
            self.num_retries = 1

        @arv_retry.retry_method
        def check(self, a, num_retries=None, z=0):
            return (a, num_retries, z)


    def test_positional_arg_raises(self):
        # unsupported use -- make sure we raise rather than ignore
        with self.assertRaises(TypeError):
            self.assertEqual((3, 2, 0), self.Tester().check(3, 2))

    def test_keyword_arg_passed(self):
        self.assertEqual((4, 3, 0), self.Tester().check(num_retries=3, a=4))

    def test_not_specified(self):
        self.assertEqual((0, 1, 0), self.Tester().check(0))

    def test_not_specified_with_other_kwargs(self):
        self.assertEqual((1, 1, 1), self.Tester().check(1, z=1))

    def test_bad_call(self):
        with self.assertRaises(TypeError):
            self.Tester().check(num_retries=2)


if __name__ == '__main__':
    unittest.main()
