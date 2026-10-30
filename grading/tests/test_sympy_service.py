import subprocess
import json
import sys
from pathlib import Path

SERVICE_PATH = Path(__file__).resolve().parent.parent / "sympy_service.py"


def grade_via_service(expected: str, answer: str, timeout: float = 10.0) -> dict:
    proc = subprocess.Popen(
        [sys.executable, str(SERVICE_PATH)],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    req = json.dumps({"id": "1", "expected": expected, "answer": answer})
    out, err = proc.communicate(input=req + "\n", timeout=timeout)
    assert not err, f"stderr: {err}"
    return json.loads(out.strip())


def test_equivalent_polynomials():
    r = grade_via_service("x^2+2x+1", "(x+1)^2")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_equivalent_expanded():
    r = grade_via_service("(x+3)(x+4)", "x^2+7x+12")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_equivalent_with_implicit_mult():
    r = grade_via_service("6x^2", "6*x^2")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_eulers_number():
    r = grade_via_service("e^x", "exp(x)")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_eulers_number_chain():
    r = grade_via_service("2e^(2x)", "2*exp(2*x)")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_trig_identity():
    r = grade_via_service("sin(x)^2+cos(x)^2", "1")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_chain_rule_derivative():
    r = grade_via_service("3(2x+1)^2", "12x^2+12x+3")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_absolute_value_with_log():
    r = grade_via_service(
        "(1/2)ln|x-1|-(1/2)ln|x+1|",
        "(1/2)*log(Abs(x-1))-(1/2)*log(Abs(x+1))",
    )
    assert r["correct"] is True, f"Expected True, got {r}"


def test_literal_equation():
    r = grade_via_service("y = (z - 2x)/3", "(z-2*x)/3")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_constant_of_integration():
    r = grade_via_service("xe^x-e^x+C", "e^x*(x-1)+C")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_plus_c_mismatch():
    r = grade_via_service("xe^x-e^x+C", "xe^x-e^x")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_not_equivalent():
    r = grade_via_service("2x+3", "2x+4")
    assert r["correct"] is False, f"Expected False, got {r}"


def test_parse_error():
    r = grade_via_service("x^2", "invalid(((")
    assert r["correct"] is False
    assert "Parse error" in r["feedback"]


def test_multiple_requests():
    proc = subprocess.Popen(
        [sys.executable, str(SERVICE_PATH)],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    requests = [
        {"id": "1", "expected": "x^2+2x+1", "answer": "(x+1)^2"},
        {"id": "2", "expected": "2x+3", "answer": "2x+4"},
        {"id": "3", "expected": "cos(x)", "answer": "cos(x)"},
    ]
    # communicate() writes the input and closes stdin itself; closing it first
    # makes Python 3.11's communicate() flush a closed pipe (ValueError).
    payload = "".join(json.dumps(req) + "\n" for req in requests)
    out, _ = proc.communicate(input=payload, timeout=10)
    results = [json.loads(line) for line in out.strip().splitlines()]
    assert len(results) == 3
    assert results[0]["correct"] is True
    assert results[1]["correct"] is False
    assert results[2]["correct"] is True


def test_equivalent_different_forms():
    r = grade_via_service("x(x+1)", "x^2+x")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_trig_double_angle():
    r = grade_via_service("sin(2x)", "2sin(x)cos(x)")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_trig_pythagorean_variant():
    r = grade_via_service("1-cos(x)^2", "sin(x)^2")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_constant_integration_c1():
    r = grade_via_service("xe^x-e^x+C_1", "e^x*(x-1)+C_2")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_constant_integration_minus_c():
    r = grade_via_service("x^2/2-C", "x^2/2+C")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_log_absolute_equivalence():
    r = grade_via_service("ln|x|", "log(Abs(x))")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_pi_symbol():
    r = grade_via_service("2pi", "2*pi")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_arcsin_variants():
    r = grade_via_service("arcsin(x)", "asin(x)")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_rational_expression_simplification():
    r = grade_via_service("(x^2-1)/(x-1)", "x+1")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_power_simplification():
    r = grade_via_service("x^(1/2)", "sqrt(x)")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_factored_vs_expanded():
    r = grade_via_service("x^2-4", "(x+2)(x-2)")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_exp_equivalence():
    r = grade_via_service("exp(2x)", "e^(2x)")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_negative_constant_integration():
    r = grade_via_service("sin(x)+C", "sin(x)-C")
    assert r["correct"] is True, f"Expected True, got {r}"


def test_economics_letters_are_variables():
    # Q, S, N and O are SymPy objects unless the grader says otherwise.
    for expected, answer in [("2Q+5", "5+2*Q"), ("1.5Q^2+4Q+46", "46+4Q+1.5Q^2"), ("N+S", "S+N"), ("gamma+1", "1+gamma")]:
        r = grade_via_service(expected, answer)
        assert r["correct"] is True, f"{expected} vs {answer}: {r}"
    r = grade_via_service("2Q+5", "2Q+6")
    assert r["correct"] is False, r


def test_thousands_separator():
    r = grade_via_service("15x+425000", "15x+425,000")
    assert r["correct"] is True, r
    r = grade_via_service("15x+425000", "15x+42,500")
    assert r["correct"] is False, r


def test_large_values_are_compared_not_skipped():
    # Every sample of this expression on [-5, 5] is above 1e10. The check used
    # to skip them all and accept any answer.
    expected = "70x^4(2x^5+9)^6"
    assert grade_via_service(expected, "70x^4(2x^5+9)^6")["correct"] is True
    assert grade_via_service(expected, "70x^4(2x^5+9)^6+1")["correct"] is False
    assert grade_via_service(expected, "70x^4(2x^5+9)^5")["correct"] is False
