// Copyright 2021 Shift Crypto AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backend

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/digitalbitbox/bitbox-wallet-app/backend/accounts"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/coins/btc"
	coinpkg "github.com/digitalbitbox/bitbox-wallet-app/backend/coins/coin"
	keystoremock "github.com/digitalbitbox/bitbox-wallet-app/backend/keystore/mocks"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/keystore/software"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/signing"
	"github.com/digitalbitbox/bitbox02-api-go/api/firmware"
	"github.com/stretchr/testify/require"
)

const dummyMsg = "message to be signed"

func defaultParams() url.Values {
	params := url.Values{}
	params.Set("v", "0")
	params.Set("msg", dummyMsg)
	params.Set("format", "any")
	params.Set("asset", "btc")
	params.Set("callback", "http://localhost/aopp/")
	return params
}

func TestAOPPSuccess(t *testing.T) {
	// From mnemonic: wisdom minute home employ west tail liquid mad deal catalog narrow mistake
	rootKey := mustXKey("xprv9s21ZrQH143K3gie3VFLgx8JcmqZNsBcBc6vAdJrsf4bPRhx69U8qZe3EYAyvRWyQdEfz7ZpyYtL8jW2d2Lfkfh6g2zivq8JdZPQqxoxLwB")
	keystoreHelper := software.NewKeystore(rootKey)
	dummySignature := []byte(`signature`)
	ks := keystoremock.KeystoreMock{
		RootFingerprintFunc: func() ([]byte, error) {
			return []byte{0x55, 0x055, 0x55, 0x55}, nil
		},
		SupportsAccountFunc: func(coin coinpkg.Coin, meta interface{}) bool {
			switch coin.(type) {
			case *btc.Coin:
				scriptType := meta.(signing.ScriptType)
				return scriptType != signing.ScriptTypeP2PKH
			default:
				return true
			}
		},
		SupportsUnifiedAccountsFunc: func() bool {
			return true
		},
		SupportsMultipleAccountsFunc: func() bool {
			return true
		},
		CanSignMessageFunc: func(coinpkg.Code) bool {
			return true
		},
		SignBTCMessageFunc: func(message []byte, keypath signing.AbsoluteKeypath, scriptType signing.ScriptType) ([]byte, error) {
			require.Equal(t, dummyMsg, string(message))
			return dummySignature, nil
		},
		SignETHMessageFunc: func(message []byte, keypath signing.AbsoluteKeypath) ([]byte, error) {
			require.Equal(t, dummyMsg, string(message))
			return dummySignature, nil
		},
		ExtendedPublicKeyFunc: keystoreHelper.ExtendedPublicKey,
	}

	tests := []struct {
		asset       string
		coinCode    coinpkg.Code
		format      string
		address     string
		accountCode accounts.Code
		accountName string
	}{
		{
			asset:       "btc",
			coinCode:    coinpkg.CodeBTC,
			format:      "any", // defaults to p2wpkh
			address:     "bc1qxp6xr63t098rl9udlynrktq00un6vqduzjgua3",
			accountCode: "v0-55555555-btc-0",
			accountName: "Bitcoin",
		},
		{
			asset:       "btc",
			coinCode:    coinpkg.CodeBTC,
			format:      "p2wpkh",
			address:     "bc1qxp6xr63t098rl9udlynrktq00un6vqduzjgua3",
			accountCode: "v0-55555555-btc-0",
			accountName: "Bitcoin",
		},
		{
			asset:       "btc",
			coinCode:    coinpkg.CodeBTC,
			format:      "p2sh",
			address:     "3C4J3CSPSYD3ibV8u1DqqPRtfsUsSbnuPX",
			accountCode: "v0-55555555-btc-0",
			accountName: "Bitcoin",
		},
		{
			asset:       "eth",
			coinCode:    coinpkg.CodeETH,
			format:      "any",
			address:     "0xB7C853464BE7Ae39c366C9C2A9D4b95340a708c7",
			accountCode: "v0-55555555-eth-0",
			accountName: "Ethereum",
		},
	}

	for _, test := range tests {
		test := test
		t.Run("", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t,
					[]string{"application/json"},
					r.Header["Content-Type"],
				)
				body, err := ioutil.ReadAll(r.Body)
				require.NoError(t, err)

				require.JSONEq(t,
					fmt.Sprintf(`{"version": 0, "address": "%s", "signature": "c2lnbmF0dXJl"}`, test.address),
					string(body),
				)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			b := newBackend(t, testnetDisabled, regtestDisabled)
			defer b.Close()

			callback := server.URL
			params := defaultParams()
			params.Set("asset", test.asset)
			params.Set("format", test.format)
			params.Set("callback", callback)

			callbackURL, err := url.Parse(callback)
			require.NoError(t, err)
			callbackHost := callbackURL.Host

			require.Equal(t, AOPP{State: aoppStateInactive}, b.AOPP())
			b.HandleURI("aopp:?" + params.Encode())
			require.Equal(t,
				AOPP{
					State:        aoppStateUserApproval,
					CallbackHost: callbackHost,
					coinCode:     test.coinCode,
					format:       test.format,
					message:      dummyMsg,
					callback:     callback,
				},
				b.AOPP(),
			)

			b.AOPPApprove()
			require.Equal(t,
				AOPP{
					State:        aoppStateAwaitingKeystore,
					CallbackHost: callbackHost,
					coinCode:     test.coinCode,
					format:       test.format,
					message:      dummyMsg,
					callback:     callback,
				},
				b.AOPP(),
			)

			b.registerKeystore(&ks)

			require.Equal(t,
				AOPP{
					State:        aoppStateChoosingAccount,
					Accounts:     []account{{Name: test.accountName, Code: test.accountCode}},
					CallbackHost: callbackHost,
					coinCode:     test.coinCode,
					format:       test.format,
					message:      dummyMsg,
					callback:     callback,
				},
				b.AOPP(),
			)

			b.AOPPChooseAccount(test.accountCode)
			require.Equal(t,
				AOPP{
					State:        aoppStateSuccess,
					Accounts:     []account{{Name: test.accountName, Code: test.accountCode}},
					Address:      test.address,
					CallbackHost: callbackHost,
					coinCode:     test.coinCode,
					format:       test.format,
					message:      dummyMsg,
					callback:     callback,
				},
				b.AOPP(),
			)
		})
	}

	// If a keystore is already registered, the user is first asked for approval to continue.
	t.Run("user-approve", func(t *testing.T) {
		b := newBackend(t, testnetDisabled, regtestDisabled)
		defer b.Close()
		params := defaultParams()
		b.registerKeystore(&ks)
		b.HandleURI("aopp:?" + params.Encode())
		require.Equal(t, aoppStateUserApproval, b.AOPP().State)
		b.AOPPApprove()
		require.Equal(t, aoppStateChoosingAccount, b.AOPP().State)
	})
	// If a keystore is already registered, the user is first asked for approval to continue. Edge
	// case: keystore is disconnected during approval.
	t.Run("user-approve-2", func(t *testing.T) {
		b := newBackend(t, testnetDisabled, regtestDisabled)
		defer b.Close()
		params := defaultParams()
		b.registerKeystore(&ks)
		b.HandleURI("aopp:?" + params.Encode())
		require.Equal(t, aoppStateUserApproval, b.AOPP().State)
		b.DeregisterKeystore()
		b.AOPPApprove()
		require.Equal(t, aoppStateAwaitingKeystore, b.AOPP().State)
	})
}

func TestAOPPFailures(t *testing.T) {
	// From mnemonic: wisdom minute home employ west tail liquid mad deal catalog narrow mistake
	rootKey := mustXKey("xprv9s21ZrQH143K3gie3VFLgx8JcmqZNsBcBc6vAdJrsf4bPRhx69U8qZe3EYAyvRWyQdEfz7ZpyYtL8jW2d2Lfkfh6g2zivq8JdZPQqxoxLwB")
	keystoreHelper := software.NewKeystore(rootKey)
	dummySignature := []byte(`signature`)
	ks := keystoremock.KeystoreMock{
		RootFingerprintFunc: func() ([]byte, error) {
			return []byte{0x55, 0x055, 0x55, 0x55}, nil
		},
		SupportsAccountFunc: func(coin coinpkg.Coin, meta interface{}) bool {
			switch coin.(type) {
			case *btc.Coin:
				scriptType := meta.(signing.ScriptType)
				return scriptType != signing.ScriptTypeP2PKH
			default:
				return true
			}
		},
		SupportsUnifiedAccountsFunc: func() bool {
			return true
		},
		SupportsMultipleAccountsFunc: func() bool {
			return true
		},
		CanSignMessageFunc: func(coinpkg.Code) bool {
			return true
		},
		SignBTCMessageFunc: func(message []byte, keypath signing.AbsoluteKeypath, scriptType signing.ScriptType) ([]byte, error) {
			require.Equal(t, dummyMsg, string(message))
			return dummySignature, nil
		},
		SignETHMessageFunc: func(message []byte, keypath signing.AbsoluteKeypath) ([]byte, error) {
			require.Equal(t, dummyMsg, string(message))
			return dummySignature, nil
		},
		ExtendedPublicKeyFunc: keystoreHelper.ExtendedPublicKey,
	}

	t.Run("wrong_version", func(t *testing.T) {
		b := newBackend(t, testnetDisabled, regtestDisabled)
		defer b.Close()
		params := defaultParams()
		params.Set("v", "1")
		b.HandleURI("aopp:?" + params.Encode())
		require.Equal(t, aoppStateError, b.AOPP().State)
		require.Equal(t, errAOPPVersion, b.AOPP().ErrorCode)

	})
	t.Run("missing_callback", func(t *testing.T) {
		b := newBackend(t, testnetDisabled, regtestDisabled)
		defer b.Close()
		params := defaultParams()
		params.Del("callback")
		b.HandleURI("aopp:?" + params.Encode())
		require.Equal(t, aoppStateError, b.AOPP().State)
		require.Equal(t, errAOPPInvalidRequest, b.AOPP().ErrorCode)
	})
	t.Run("invalid_callback", func(t *testing.T) {
		b := newBackend(t, testnetDisabled, regtestDisabled)
		defer b.Close()
		params := defaultParams()
		params.Set("callback", ":not a valid url")
		b.HandleURI("aopp:?" + params.Encode())
		require.Equal(t, aoppStateError, b.AOPP().State)
		require.Equal(t, errAOPPInvalidRequest, b.AOPP().ErrorCode)
	})
	t.Run("missing_msg", func(t *testing.T) {
		b := newBackend(t, testnetDisabled, regtestDisabled)
		defer b.Close()
		params := defaultParams()
		params.Del("msg")
		b.HandleURI("aopp:?" + params.Encode())
		require.Equal(t, aoppStateError, b.AOPP().State)
		require.Equal(t, errAOPPInvalidRequest, b.AOPP().ErrorCode)
	})
	t.Run("unsupported_asset", func(t *testing.T) {
		b := newBackend(t, testnetDisabled, regtestDisabled)
		defer b.Close()
		params := defaultParams()
		params.Set("asset", "<invalid>")
		b.HandleURI("aopp:?" + params.Encode())
		require.Equal(t, aoppStateError, b.AOPP().State)
		require.Equal(t, errAOPPUnsupportedAsset, b.AOPP().ErrorCode)
	})
	t.Run("cant_sign", func(t *testing.T) {
		b := newBackend(t, testnetDisabled, regtestDisabled)
		defer b.Close()
		params := defaultParams()
		b.HandleURI("aopp:?" + params.Encode())
		b.AOPPApprove()
		ks2 := ks
		ks2.CanSignMessageFunc = func(coinpkg.Code) bool {
			return false
		}
		b.registerKeystore(&ks2)
		require.Equal(t, aoppStateError, b.AOPP().State)
		require.Equal(t, errAOPPUnsupportedKeystore, b.AOPP().ErrorCode)
	})
	t.Run("no_accounts", func(t *testing.T) {
		b := newBackend(t, testnetDisabled, regtestDisabled)
		defer b.Close()
		params := defaultParams()
		b.registerKeystore(&ks)
		require.NoError(t, b.SetAccountActive("v0-55555555-btc-0", false))
		b.HandleURI("aopp:?" + params.Encode())
		b.AOPPApprove()
		require.Equal(t, aoppStateError, b.AOPP().State)
		require.Equal(t, errAOPPNoAccounts, b.AOPP().ErrorCode)
	})
	t.Run("unsupported_format", func(t *testing.T) {
		b := newBackend(t, testnetDisabled, regtestDisabled)
		defer b.Close()
		params := defaultParams()
		params.Set("format", "p2pkh")
		b.HandleURI("aopp:?" + params.Encode())
		b.AOPPApprove()
		b.registerKeystore(&ks)
		require.Equal(t, aoppStateError, b.AOPP().State)
		require.Equal(t, errAOPPUnsupportedFormat, b.AOPP().ErrorCode)
	})
	t.Run("signing_aborted", func(t *testing.T) {
		b := newBackend(t, testnetDisabled, regtestDisabled)
		defer b.Close()
		params := defaultParams()
		b.HandleURI("aopp:?" + params.Encode())
		b.AOPPApprove()
		ks2 := ks
		ks2.SignBTCMessageFunc = func([]byte, signing.AbsoluteKeypath, signing.ScriptType) ([]byte, error) {
			return nil, firmware.NewError(firmware.ErrUserAbort, "")
		}
		b.registerKeystore(&ks2)
		b.AOPPChooseAccount("v0-55555555-btc-0")
		require.Equal(t, aoppStateError, b.AOPP().State)
		require.Equal(t, errAOPPSigningAborted, b.AOPP().ErrorCode)
	})
	t.Run("callback_failed", func(t *testing.T) {
		b := newBackend(t, testnetDisabled, regtestDisabled)
		defer b.Close()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		params := defaultParams()
		params.Set("callback", server.URL)
		b.HandleURI("aopp:?" + params.Encode())
		b.AOPPApprove()
		b.registerKeystore(&ks)
		b.AOPPChooseAccount("v0-55555555-btc-0")
		require.Equal(t, aoppStateError, b.AOPP().State)
		require.Equal(t, errAOPPCallback, b.AOPP().ErrorCode)
	})
}
