import { useEffect, useState } from 'react';

const ENCOURAGEMENTS = [
  'Today is a good day to build something solid.',
  'Good work takes a little time. This graph is worth the wait.',
  'Your repository is getting its map drawn carefully.',
  'A few more moments, and the structure will start making sense.',
  'Big codebases take patience. You are doing fine.',
  'The tool is reading the details so you do not have to.',
  'Every large graph starts with one parsed file.',
  'This is a quiet moment before the codebase opens up.',
  'Thanks for waiting. The local engine is still working.',
  'You chose the repo. AVmatrix is building the view.',
  'You are doing better than you think.',
  'A small pause today can become a clear answer in a moment.',
  'Good things are still happening while the spinner turns.',
  'Keep going. The next useful view is getting closer.',
  'Some days need patience. This one also has progress.',
  'You brought the code. AVmatrix is bringing the map.',
  'One steady step is still a step forward.',
  'The best work often looks quiet while it is being built.',
  'Stay with it. Useful structure is taking shape.',
  'A little waiting is easier when the result is worth opening.',
];

const ENCOURAGEMENT_COLORS = [
  'text-red-500',
  'text-orange-500',
  'text-amber-500',
  'text-yellow-500',
  'text-lime-500',
  'text-green-500',
  'text-teal-500',
  'text-cyan-500',
  'text-blue-500',
  'text-violet-500',
  'text-fuchsia-500',
  'text-pink-500',
  'text-rose-500',
];

const randomIndex = (length: number, previous?: number): number => {
  if (length <= 1) return 0;
  let next = Math.floor(Math.random() * length);
  if (next === previous) {
    next = (next + 1) % length;
  }
  return next;
};

export const EncouragementLine = () => {
  const [messageIndex, setMessageIndex] = useState(() => randomIndex(ENCOURAGEMENTS.length));
  const [colorIndex, setColorIndex] = useState(() => randomIndex(ENCOURAGEMENT_COLORS.length));

  useEffect(() => {
    const timer = window.setInterval(() => {
      setMessageIndex((previous) => randomIndex(ENCOURAGEMENTS.length, previous));
      setColorIndex((previous) => randomIndex(ENCOURAGEMENT_COLORS.length, previous));
    }, 7_000);

    return () => window.clearInterval(timer);
  }, []);

  return (
    <p
      className={`mx-auto min-h-[3rem] max-w-2xl px-4 text-center font-reading text-sm leading-6 ${ENCOURAGEMENT_COLORS[colorIndex]}`}
      aria-live="polite"
    >
      {ENCOURAGEMENTS[messageIndex]}
    </p>
  );
};
