import { VERBOSE } from '../constants/config';

export const log = (...args: unknown[]) => VERBOSE && console.log(...args);
