// File starts with 'a-' to sort alphabetically before 'b-provider.ts'.
// Resolution must not depend on provider files being parsed first.
import { getUser } from './b-provider';

export function main() {
  const x = getUser();
  x.save();
}
