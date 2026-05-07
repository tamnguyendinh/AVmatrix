// File starts with 'a-' to sort alphabetically before 'b-provider.js'.
// Resolution must not depend on provider files being parsed first.
import { getUser } from './b-provider';

export function main() {
  const u = getUser();
  u.save();
}
