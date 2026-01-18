import chalk from 'chalk';
import { VERBOSE } from '../constants/config';

export const logVerbose = (...args: unknown[]) => VERBOSE && console.log(...args);

export const logError = (...args: unknown[]) => console.error(chalk.red(...args));
export const logWarn = (...args: unknown[]) => console.warn(chalk.yellow(...args));
export const logInfo = (...args: unknown[]) => console.log(chalk.blue(...args));
export const logSuccess = (...args: unknown[]) => console.log(chalk.green(...args));
