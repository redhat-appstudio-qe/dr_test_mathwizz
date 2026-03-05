// Math solver page component
// Uses CalculatorComponent for the main interface

import React, { useState } from 'react';
import { solve } from '../api';
import { validateProblem } from '../utils';
import CalculatorComponent from './CalculatorComponent';

const SolverPage = () => {
  const [problem, setProblem] = useState('');
  const [answer, setAnswer] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSolve = async () => {
    setError('');
    setAnswer('');

    const validationError = validateProblem(problem);
    if (validationError) {
      setError(validationError);
      return;
    }

    setLoading(true);

    try {
      const result = await solve(problem);
      setAnswer(result.answer);
      setProblem('');
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="page-container">
      <CalculatorComponent
        problem={problem}
        answer={answer}
        onProblemChange={setProblem}
        onSolve={handleSolve}
        loading={loading}
        error={error}
      />
    </div>
  );
};

export default SolverPage;
