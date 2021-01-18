// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

import (
	"sync/atomic"
	"unsafe"
)

//goland:noinspection GoSnakeCaseUsage
const (
	/*
	TODO: comment
	*/
	_LLS_STANDBY uint32 = 0
	_LLS_SOURCE_PENDING uint32 = 1
	_LLS_LOAD_PENDING uint32 = 2
	_LLS_READY uint32 = 10
)

var (
	/*

	*/
	defaultClient Client
)

/*
TODO: comment
*/
func (c *Client) isValid() bool {
	return c != nil
}

/*
TODO: comment
*/
func (c *Client) changeState(from, to uint32) bool {
	return atomic.CompareAndSwapUint32(&c.state, from, to)
}

/*
TODO: comment
*/
func (c *Client) changeStateForce(to uint32) {
	atomic.StoreUint32(&c.state, to)
}

/*
TODO: comment
*/
func (c *Client) getState() uint32 {
	return atomic.LoadUint32(&c.state)
}

/*
TODO: comment
*/
func strState(v uint32) string {
	switch v {
	case _LLS_STANDBY:
		return "<standby mode>"
	case _LLS_SOURCE_PENDING:
		return "<analyzing locale sources>"
	case _LLS_LOAD_PENDING:
		return "<loading locales>"
	case _LLS_READY:
		return "<locales loaded, ready to use>"
	default:
		return "<unknown>"
	}
}

/*
getDefaultLocale returns a Locale object that was marked as default locale.

If either no one Locale object was marked as default
or no one locale was loaded yet, nil is returned.
*/
func (c *Client) getDefaultLocale() *Locale {
	if c.getState() != _LLS_READY {
		return nil
	}
	return (*Locale)(atomic.LoadPointer(&c.defaultLocale))
}

/*
setDefaultLocale marks loc as a default locale saving it to the defaultLocale
atomically.
*/
func (c *Client) setDefaultLocale(loc *Locale) {
	atomic.StorePointer(&c.defaultLocale, unsafe.Pointer(loc))
}

/*
getLocale returns a Locale object for the requested locale's name.
E.g: It returns English locale's entry point if you passed "en_US" and you did load
locale with that name.

If either Locale with the requested name is not exist,
or no one locale was loaded yet nil is returned.
*/
func (c *Client) getLocale(name string) *Locale {
	if c.getState() != _LLS_READY {
		return nil
	}
	return c.storage[name]
}

/*
makeLocale is Locale constructor and initializer.
The caller MUST to add it to either Client.storage or Client.storageTmp
depends on requirements.
*/
func (c *Client) makeLocale(name string) *Locale {
	loc := &Locale{
		owner: c,
		name:  name,
	}

	loc.root = loc.makeSubNode()
	return loc
}