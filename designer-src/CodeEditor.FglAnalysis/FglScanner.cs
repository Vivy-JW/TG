using System;
using System.Text;

namespace CodeEditor.FglAnalysis
{
	// Token: 0x02000006 RID: 6
	public class FglScanner
	{
		// Token: 0x06000015 RID: 21 RVA: 0x00002153 File Offset: 0x00000353
		public FglScanner(FglReader reader)
		{
			this._reader = reader;
			this._currentToken = new FglToken();
			this._line = 0;
			this._state = FglScanner.ScanState.START_STATE;
		}

		// Token: 0x06000016 RID: 22 RVA: 0x00002188 File Offset: 0x00000388
		private FglToken GetKeywordToken()
		{
			string text;
			switch (text = this.bufferStr.ToUpper())
			{
			case "DISPLAY":
				return this.MakeToken(TokenType.DISPLAY_TOKEN, this.bufferStr);
			case "DIALOG":
				return this.MakeToken(TokenType.DIALOG_TOKEN, this.bufferStr);
			case "INPUT":
				return this.MakeToken(TokenType.INPUT_TOKEN, this.bufferStr);
			case "BY":
				return this.MakeToken(TokenType.BY_TOKEN, this.bufferStr);
			case "NAME":
				return this.MakeToken(TokenType.NAME_TOKEN, this.bufferStr);
			case "ARRAY":
				return this.MakeToken(TokenType.ARRAY_TOKEN, this.bufferStr);
			case "END":
				return this.MakeToken(TokenType.END_TOKEN, this.bufferStr);
			case "ON":
				return this.MakeToken(TokenType.ON_TOKEN, this.bufferStr);
			case "ACTION":
				return this.MakeToken(TokenType.ACTION_TOKEN, this.bufferStr);
			case "BEFORE":
				return this.MakeToken(TokenType.BEFORE_TOKEN, this.bufferStr);
			case "AFTER":
				return this.MakeToken(TokenType.AFTER_TOKEN, this.bufferStr);
			case "FIELD":
				return this.MakeToken(TokenType.FIELD_TOKEN, this.bufferStr);
			case "FUNCTION":
				return this.MakeToken(TokenType.FUNCTION_TOKEN, this.bufferStr);
			case "TO":
				return this.MakeToken(TokenType.TO_TOKEN, this.bufferStr);
			case "FROM":
				return this.MakeToken(TokenType.FROM_TOKEN, this.bufferStr);
			case "WITHOUT":
				return this.MakeToken(TokenType.WITHOUT_TOKEN, this.bufferStr);
			case "DEFAULTS":
				return this.MakeToken(TokenType.DEFAULTS_TOKEN, this.bufferStr);
			case "CONSTRUCT":
				return this.MakeToken(TokenType.CONSTRUCT_TOKEN, this.bufferStr);
			case "ROW":
				return this.MakeToken(TokenType.ROW_TOKEN, this.bufferStr);
			case "INSERT":
				return this.MakeToken(TokenType.INSERT_TOKEN, this.bufferStr);
			case "DELETE":
				return this.MakeToken(TokenType.DELETE_TOKEN, this.bufferStr);
			case "MAIN":
				return this.MakeToken(TokenType.MAIN_TOKEN, this.bufferStr);
			case "ATTRIBUTES":
				return this.MakeToken(TokenType.ATTRIBUTES_TOKEN, this.bufferStr);
			case "ATTRIBUTE":
				return this.MakeToken(TokenType.ATTRIBUTE_TOKEN, this.bufferStr);
			case "IF":
				return this.MakeToken(TokenType.IF_TOKEN, this.bufferStr);
			case "INFIELD":
				return this.MakeToken(TokenType.INFIELD_TOKEN, this.bufferStr);
			case "CHANGE":
				return this.MakeToken(TokenType.CHANGE_TOKEN, this.bufferStr);
			case "PUBLIC":
				return this.MakeToken(TokenType.PUBLIC_TOKEN, this.bufferStr);
			case "PRIVATE":
				return this.MakeToken(TokenType.PRIVATE_TOKEN, this.bufferStr);
			case "CALL":
				return this.MakeToken(TokenType.CALL_TOKEN, this.bufferStr);
			case "ACCEPT":
				return this.MakeToken(TokenType.ACCEPT_TOKEN, this.bufferStr);
			case "EXIT":
				return this.MakeToken(TokenType.EXIT_TOKEN, this.bufferStr);
			case "CONTINUE":
				return this.MakeToken(TokenType.CONTINUE_TOKEN, this.bufferStr);
			case "OPTIONS":
				return this.MakeToken(TokenType.OPTIONS_TOKEN, this.bufferStr);
			case "ORDER":
				return this.MakeToken(TokenType.ORDER_TOKEN, this.bufferStr);
			case "FORM":
				return this.MakeToken(TokenType.FORM_TOKEN, this.bufferStr);
			case "CONSTRAINED":
				return this.MakeToken(TokenType.CONSTRAINED_TOKEN, this.bufferStr);
			case "NO":
				return this.MakeToken(TokenType.NO_TOKEN, this.bufferStr);
			case "WRAP":
				return this.MakeToken(TokenType.WRAP_TOKEN, this.bufferStr);
			case "START":
				return this.MakeToken(TokenType.START_TOKEN, this.bufferStr);
			case "REPORT":
				return this.MakeToken(TokenType.REPORT_TOKEN, this.bufferStr);
			case "OUTPUT":
				return this.MakeToken(TokenType.OUTPUT_TOKEN, this.bufferStr);
			case "FORMAT":
				return this.MakeToken(TokenType.FORMAT_TOKEN, this.bufferStr);
			case "FIRST":
				return this.MakeToken(TokenType.FIRST_TOKEN, this.bufferStr);
			case "PAGE":
				return this.MakeToken(TokenType.PAGE_TOKEN, this.bufferStr);
			case "HEADER":
				return this.MakeToken(TokenType.HEADER_TOKEN, this.bufferStr);
			case "GROUP":
				return this.MakeToken(TokenType.GROUP_TOKEN, this.bufferStr);
			case "OF":
				return this.MakeToken(TokenType.OF_TOKEN, this.bufferStr);
			case "EVERY":
				return this.MakeToken(TokenType.EVERY_TOKEN, this.bufferStr);
			case "FINISH":
				return this.MakeToken(TokenType.FINISH_TOKEN, this.bufferStr);
			case "TRAILER":
				return this.MakeToken(TokenType.TRAILER_TOKEN, this.bufferStr);
			case "LAST":
				return this.MakeToken(TokenType.LAST_TOKEN, this.bufferStr);
			case "LET":
				return this.MakeToken(TokenType.LET_TOKEN, this.bufferStr);
			case "RUN":
				return this.MakeToken(TokenType.RUN_TOKEN, this.bufferStr);
			case "FOR":
				return this.MakeToken(TokenType.FOR_TOEKN, this.bufferStr);
			case "FOREACH":
				return this.MakeToken(TokenType.FOREACH_TOKEN, this.bufferStr);
			case "WHILE":
				return this.MakeToken(TokenType.WHILE_TOKEN, this.bufferStr);
			case "MENU":
				return this.MakeToken(TokenType.MENU_TOKEN, this.bufferStr);
			case "WHENEVER":
				return this.MakeToken(TokenType.WHENEVER_TOKEN, this.bufferStr);
			case "ANY":
				return this.MakeToken(TokenType.ANY_TOKEN, this.bufferStr);
			case "ERROR":
				return this.MakeToken(TokenType.ERROR_TOKEN, this.bufferStr);
			case "SQLERROR":
				return this.MakeToken(TokenType.SQLERROR_TOKEN, this.bufferStr);
			case "NOT":
				return this.MakeToken(TokenType.NOT_TOKEN, this.bufferStr);
			case "FOUND":
				return this.MakeToken(TokenType.FOUND_TOKEN, this.bufferStr);
			case "WARNING":
				return this.MakeToken(TokenType.WARNING_TOKEN, this.bufferStr);
			case "STOP":
				return this.MakeToken(TokenType.STOP_TOKEN, this.bufferStr);
			case "RAISE":
				return this.MakeToken(TokenType.RAISE_TOKEN, this.bufferStr);
			case "GOTO":
				return this.MakeToken(TokenType.GOTO_TOKEN, this.bufferStr);
			}
			return null;
		}

		// Token: 0x06000017 RID: 23 RVA: 0x00002A58 File Offset: 0x00000C58
		public FglToken NextToken()
		{
			for (;;)
			{
				switch (this._state)
				{
				case FglScanner.ScanState.START_STATE:
				{
					char c = this._reader.NextChar();
					if (char.IsLetterOrDigit(c))
					{
						this.bufferStr += c;
					}
					else
					{
						char c2 = c;
						if (c2 > '\r')
						{
							switch (c2)
							{
							case ' ':
								goto IL_00DC;
							case '!':
							case '$':
							case '%':
							case '&':
							case '*':
							case '+':
								goto IL_036C;
							case '"':
							case '\'':
								if (this.bufferStr.Length > 0)
								{
									goto Block_10;
								}
								this.SkipConstant(c);
								continue;
							case '#':
								break;
							case '(':
								goto IL_0217;
							case ')':
								goto IL_025B;
							case ',':
								goto IL_02E3;
							case '-':
								if (this.bufferStr.Length > 0 && this.bufferStr[this.bufferStr.Length - 1] == '-')
								{
									this.SkipMemos(c);
									continue;
								}
								this.bufferStr += c;
								continue;
							case '.':
								if (this.bufferStr.Length > 0)
								{
									this.bufferStr += c;
									continue;
								}
								goto IL_01D0;
							default:
								if (c2 == ':')
								{
									goto IL_029F;
								}
								if (c2 != '{')
								{
									goto IL_036C;
								}
								break;
							}
							this.SkipMemos(c);
							break;
						}
						if (c2 != '\0')
						{
							switch (c2)
							{
							case '\t':
								break;
							case '\n':
								this._line++;
								if (this.bufferStr.Length > 0)
								{
									goto Block_15;
								}
								continue;
							case '\v':
							case '\f':
								goto IL_036C;
							case '\r':
								continue;
							default:
								goto IL_036C;
							}
						}
						else
						{
							if (this.bufferStr.Length > 0)
							{
								goto Block_25;
							}
							this._state = FglScanner.ScanState.END_STATE;
							break;
						}
						IL_00DC:
						if (this.bufferStr.Length > 0)
						{
							goto Block_8;
						}
						break;
						IL_036C:
						this.bufferStr += c;
					}
					break;
				}
				case FglScanner.ScanState.END_STATE:
					goto IL_0019;
				}
			}
			IL_0019:
			return this.MakeToken(TokenType.EOS_TOKEN);
			Block_8:
			FglToken keywordToken = this.GetKeywordToken();
			if (keywordToken == null)
			{
				return this.MakeToken(TokenType.UNKNOWN_TOKEN, this.bufferStr);
			}
			return keywordToken;
			Block_10:
			this._reader.LookBack();
			FglToken keywordToken2 = this.GetKeywordToken();
			if (keywordToken2 == null)
			{
				return this.MakeToken(TokenType.UNKNOWN_TOKEN, this.bufferStr);
			}
			return keywordToken2;
			IL_01D0:
			return this.MakeToken(TokenType.DOT_TOKEN, ".");
			Block_15:
			FglToken keywordToken3 = this.GetKeywordToken();
			if (keywordToken3 == null)
			{
				return this.MakeToken(TokenType.UNKNOWN_TOKEN, this.bufferStr);
			}
			return keywordToken3;
			IL_0217:
			if (this.bufferStr.Length <= 0)
			{
				return this.MakeToken(TokenType.LEFTPAREN_TOKEN, "(");
			}
			this._reader.LookBack();
			FglToken keywordToken4 = this.GetKeywordToken();
			if (keywordToken4 == null)
			{
				return this.MakeToken(TokenType.UNKNOWN_TOKEN, this.bufferStr);
			}
			return keywordToken4;
			IL_025B:
			if (this.bufferStr.Length <= 0)
			{
				return this.MakeToken(TokenType.RIGHTPAREN_TOKEN, ")");
			}
			this._reader.LookBack();
			FglToken keywordToken5 = this.GetKeywordToken();
			if (keywordToken5 == null)
			{
				return this.MakeToken(TokenType.UNKNOWN_TOKEN, this.bufferStr);
			}
			return keywordToken5;
			IL_029F:
			if (this.bufferStr.Length <= 0)
			{
				return this.MakeToken(TokenType.COLON_TOKEN, ":");
			}
			this._reader.LookBack();
			FglToken keywordToken6 = this.GetKeywordToken();
			if (keywordToken6 == null)
			{
				return this.MakeToken(TokenType.UNKNOWN_TOKEN, this.bufferStr);
			}
			return keywordToken6;
			IL_02E3:
			if (this.bufferStr.Length <= 0)
			{
				return this.MakeToken(TokenType.PEROID_TOKEN, ",");
			}
			this._reader.CurrPos = this._reader.CurrPos - 1;
			FglToken keywordToken7 = this.GetKeywordToken();
			if (keywordToken7 == null)
			{
				return this.MakeToken(TokenType.UNKNOWN_TOKEN, this.bufferStr);
			}
			return keywordToken7;
			Block_25:
			FglToken keywordToken8 = this.GetKeywordToken();
			if (keywordToken8 == null)
			{
				return this.MakeToken(TokenType.UNKNOWN_TOKEN, this.bufferStr);
			}
			return keywordToken8;
		}

		// Token: 0x1700000A RID: 10
		// (get) Token: 0x06000018 RID: 24 RVA: 0x00002DEC File Offset: 0x00000FEC
		public FglToken PreviousToken
		{
			get
			{
				return this._previousToken;
			}
		}

		// Token: 0x06000019 RID: 25 RVA: 0x00002DF4 File Offset: 0x00000FF4
		private void SkipMemos(char c)
		{
			do
			{
				c = this._reader.NextChar();
				if ('\n' == c)
				{
					this._line++;
				}
			}
			while (c != '\r' && c != '\n' && c != '\0');
		}

		// Token: 0x0600001A RID: 26 RVA: 0x00002E23 File Offset: 0x00001023
		private void SkipAreaMemos(char c)
		{
			do
			{
				c = this._reader.NextChar();
				if ('\n' == c)
				{
					this._line++;
				}
			}
			while (c != '}');
		}

		// Token: 0x0600001B RID: 27 RVA: 0x00002E4C File Offset: 0x0000104C
		private void SkipConstant(char c)
		{
			char c2 = c;
			if ('\n' == c)
			{
				this._line++;
			}
			int currPos = this._reader.CurrPos;
			string text = c2.ToString();
			StringBuilder stringBuilder = new StringBuilder(c2.ToString());
			string text3;
			for (;;)
			{
				c = this._reader.NextChar();
				if (c == '\0')
				{
					break;
				}
				if ('\n' == c)
				{
					this._line++;
				}
				if ('\\' == c)
				{
					string text2 = c.ToString();
					c = this._reader.NextChar();
					text2 += c;
					stringBuilder.Append(text2.ToString());
					if (c == c2)
					{
						continue;
					}
				}
				stringBuilder.Append(c);
				text3 = stringBuilder.ToString(stringBuilder.Length - 2, 2);
				if (!(text3 == text) && c == c2)
				{
					goto Block_6;
				}
			}
			return;
			Block_6:
			stringBuilder.Clear();
			text3 = string.Empty;
		}

		// Token: 0x0600001C RID: 28 RVA: 0x00002F24 File Offset: 0x00001124
		private FglToken MakeToken(TokenType type)
		{
			if (this._currentToken != null)
			{
				this._previousToken = this._currentToken.Clone();
			}
			this._currentToken.Type = type;
			this._currentToken.Text = type.ToString();
			this.bufferStr = string.Empty;
			return this._currentToken;
		}

		// Token: 0x0600001D RID: 29 RVA: 0x00002F80 File Offset: 0x00001180
		public FglToken MakeToken(TokenType type, string text)
		{
			FglToken fglToken = this.MakeToken(type);
			fglToken.Text = text;
			return fglToken;
		}

		// Token: 0x1700000B RID: 11
		// (get) Token: 0x0600001E RID: 30 RVA: 0x00002F9D File Offset: 0x0000119D
		public int LineNumber
		{
			get
			{
				return this._line;
			}
		}

		// Token: 0x0600001F RID: 31 RVA: 0x00002FA8 File Offset: 0x000011A8
		public string GetPreviousLine(int backLineCount)
		{
			if (this.LineNumber - backLineCount < 0)
			{
				return string.Empty;
			}
			if (this._lines == null)
			{
				this._lines = this._reader.Data.Split(new char[] { '\n' });
			}
			return this._lines[this.LineNumber - backLineCount].Replace("\r", "");
		}

		// Token: 0x04000011 RID: 17
		private FglReader _reader;

		// Token: 0x04000012 RID: 18
		private FglToken _previousToken;

		// Token: 0x04000013 RID: 19
		private FglToken _currentToken;

		// Token: 0x04000014 RID: 20
		private int _line;

		// Token: 0x04000015 RID: 21
		private FglScanner.ScanState _state;

		// Token: 0x04000016 RID: 22
		private string bufferStr = string.Empty;

		// Token: 0x04000017 RID: 23
		private string[] _lines;

		// Token: 0x02000007 RID: 7
		public enum ScanState
		{
			// Token: 0x04000019 RID: 25
			START_STATE,
			// Token: 0x0400001A RID: 26
			END_STATE,
			// Token: 0x0400001B RID: 27
			IDENTIFIER_STATE
		}
	}
}
