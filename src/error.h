/*
 * Copyright (C) 2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

#ifndef SNAP_CONFINE_ERROR_H
#define SNAP_CONFINE_ERROR_H

/**
 * This module defines APIs for simple error management.
 *
 * Errors are allocated objects that can be returned and passed around from
 * functions.  Errors carry a formatted message and optionally a scoped error
 * code. The code is coped with a string "domain" that simply acts as a
 * namespace for various interacting modules.
 **/

/**
 * Opaque error structure.
 **/
struct sc_error;

/**
 * Error domain for errors related to system errno.
 **/
#define SC_ERRNO_DOMAIN "errno"

/**
 * Initialize a new error object.
 *
 * The domain is a cookie-like string that allows the caller to distinguish
 * between "namespaces" of error codes. It should be a static string that is
 * provided by the caller. Both the domain and the error code can be retrieved
 * later.
 *
 * This function calls die() in case of memory allocation failure.
 **/
__attribute__ ((format(printf, 3, 4)))
struct sc_error *sc_error_init(const char *domain, int code, const char *msgfmt,
			       ...);

/**
 * Initialize an errno-based error.
 *
 * The error carries a copy of errno and a custom error message as designed by
 * the caller. See sc_error_init() for a more complete description.
 *
 * This function calls die() in case of memory allocation failure.
 **/
__attribute__ ((format(printf, 2, 3)))
struct sc_error *sc_error_init_from_errno(int errno_copy, const char *msgfmt,
					  ...);

/**
 * Get the error domain out of an error object.
 *
 * The error domain acts as a namespace for error codes. The return value must
 * not be released in any way.
 **/
const char *sc_error_domain(struct sc_error *err);

/**
 * Get the error code out of an error object.
 *
 * The error code is scoped by the error domain.
 *
 * An error code of zero is special-cased to indicate that no particular error
 * code is reserved for this error and it's not something that the programmer
 * can rely on programmatically. This can be used to return an error message
 * without having to allocate a distinct code for each one.
 **/
int sc_error_code(struct sc_error *err);

/**
 * Get the error message out of an error object.
 *
 * The error message is bound to the life-cycle of the error object.
 **/
const char *sc_error_msg(struct sc_error *err);

/**
 * Free an error object.
 *
 * The error object can be NULL.
 **/
void sc_error_free(struct sc_error *error);

/**
 * Cleanup an error with sc_error_free()
 *
 * This function is designed to be used with
 * __attribute__((cleanup(sc_cleanup_error))).
 **/
void sc_cleanup_error(struct sc_error **ptr);

/**
 *
 * Die if there's an error.
 *
 * This function is a correct way to die() if the passed error is not NULL.
 *
 * The error message is derived from the data in the error, using the special
 * errno domain to provide additional information if that is available.
 **/
void sc_error_die(struct sc_error *error);

/**
 * Forward an error to the caller.
 *
 * This tries to forward an error to the caller. If this is impossible because
 * the caller did not provide a location for the error to be stored then the
 * sc_error_die() is called as a safety measure.
 **/
void sc_error_forward(struct sc_error **recepient, struct sc_error *error);

#endif
