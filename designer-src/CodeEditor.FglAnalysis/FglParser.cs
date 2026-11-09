using System;
using System.Collections.Generic;
using System.Linq;

namespace CodeEditor.FglAnalysis
{
	// Token: 0x0200000A RID: 10
	public class FglParser
	{
		// Token: 0x0600002D RID: 45 RVA: 0x000031AA File Offset: 0x000013AA
		public FglParser(FglScanner scanner)
		{
			this._scanner = scanner;
			this._currentToken = new FglToken();
			this._predictToken = new FglToken();
			this._used = true;
		}

		// Token: 0x0600002E RID: 46 RVA: 0x000031D6 File Offset: 0x000013D6
		public FglToken Next()
		{
			if (this._used)
			{
				this._currentToken = this._scanner.NextToken();
			}
			else
			{
				this._currentToken = this._predictToken;
				this._used = true;
			}
			return this._currentToken;
		}

		// Token: 0x0600002F RID: 47 RVA: 0x0000320C File Offset: 0x0000140C
		private FglToken Predict()
		{
			if (this._used)
			{
				this._previous = this._predictToken.Clone();
				this._predictToken = this._scanner.NextToken();
				this._used = false;
			}
			return this._predictToken;
		}

		// Token: 0x06000030 RID: 48 RVA: 0x00003245 File Offset: 0x00001445
		private FglToken Previous()
		{
			return this._previous;
		}

		// Token: 0x06000031 RID: 49 RVA: 0x0000324D File Offset: 0x0000144D
		public void CancelJob()
		{
			this.isCancelled = true;
		}

		// Token: 0x06000032 RID: 50 RVA: 0x00003258 File Offset: 0x00001458
		public AST Parse()
		{
			this._root = new AST
			{
				Line = -1,
				NodeKind = 0
			};
			this._currentNode = this._root;
			while (!this.isCancelled)
			{
				TokenType type = this.Predict().Type;
				if (type <= TokenType.FUNCTION_TOKEN)
				{
					if (type == TokenType.EOS_TOKEN)
					{
						return this._root;
					}
					switch (type)
					{
					case TokenType.EXIT_TOKEN:
					case TokenType.ACCEPT_TOKEN:
					case TokenType.CONTINUE_TOKEN:
						this.ParseOthersBlock();
						continue;
					case TokenType.ATTRIBUTE_TOKEN:
					case TokenType.CALL_TOKEN:
					case TokenType.PUBLIC_TOKEN:
					case TokenType.PRIVATE_TOKEN:
					case TokenType.INFIELD_TOKEN:
					case TokenType.CHANGE_TOKEN:
					case TokenType.TO_TOKEN:
					case TokenType.BY_TOKEN:
					case TokenType.NAME_TOKEN:
					case TokenType.ARRAY_TOKEN:
					case TokenType.ACTION_TOKEN:
					case TokenType.FIELD_TOKEN:
						goto IL_018C;
					case TokenType.MAIN_TOKEN:
						this.ParseMainFunction();
						continue;
					case TokenType.DISPLAY_TOKEN:
					case TokenType.MENU_TOKEN:
					case TokenType.PROMPT_TOKEN:
					case TokenType.INPUT_TOKEN:
						break;
					case TokenType.DIALOG_TOKEN:
						if (this._currentNode.NodeKind == 0)
						{
							this.ParseReportDialogFunction();
							continue;
						}
						this.ParseInteract();
						continue;
					case TokenType.END_TOKEN:
						this.ParseEnd();
						continue;
					case TokenType.ON_TOKEN:
						this.ParseOnBlock();
						continue;
					case TokenType.BEFORE_TOKEN:
						this.ParseBeforeBlock();
						continue;
					case TokenType.AFTER_TOKEN:
						this.ParseAfterBlock();
						continue;
					case TokenType.FUNCTION_TOKEN:
						this.ParseFunction();
						continue;
					default:
						goto IL_018C;
					}
				}
				else if (type != TokenType.CONSTRUCT_TOKEN)
				{
					if (type != TokenType.REPORT_TOKEN)
					{
						if (type != TokenType.WHENEVER_TOKEN)
						{
							goto IL_018C;
						}
						this.ParseWhenever();
						continue;
					}
					else
					{
						if (this._currentNode.NodeKind == 0)
						{
							this.ParseReportDialogFunction();
							continue;
						}
						this.Next();
						continue;
					}
				}
				this.ParseInteract();
				continue;
				IL_018C:
				this.Next();
			}
			return this._root;
		}

		// Token: 0x06000033 RID: 51 RVA: 0x000033FC File Offset: 0x000015FC
		private AST CreateElement(int nodeKind)
		{
			return new AST
			{
				Line = this._scanner.LineNumber + 1,
				NodeKind = nodeKind,
				ParentNode = this._currentNode
			};
		}

		// Token: 0x06000034 RID: 52 RVA: 0x00003438 File Offset: 0x00001638
		private FunctionAST CreateFunctionElement(int nodeKind)
		{
			if (this._root.ChildNodes.Count > 0 && this._root.ChildNodes.LastOrDefault<AST>().EndNode == null)
			{
				this._currentNode = this._root.ChildNodes.LastOrDefault<AST>();
				this.Next();
				return null;
			}
			this._currentNode = this._root;
			return new FunctionAST
			{
				Line = this._scanner.LineNumber + 1,
				NodeKind = nodeKind,
				ParentNode = this._currentNode
			};
		}

		// Token: 0x06000035 RID: 53 RVA: 0x000034CC File Offset: 0x000016CC
		private void ParseFlowControl()
		{
			AST ast = null;
			bool flag = false;
			while (!flag)
			{
				if (this.isCancelled)
				{
					flag = true;
				}
				TokenType type = this.Predict().Type;
				if (type != TokenType.END_TOKEN)
				{
					switch (type)
					{
					case TokenType.FOR_TOEKN:
					case TokenType.FOREACH_TOKEN:
					case TokenType.WHILE_TOKEN:
						ast = this.CreateElement(5);
						ast.TokenNode = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
						break;
					default:
						this.Next();
						break;
					}
				}
				else
				{
					this.Next();
				}
			}
			if (ast != null)
			{
				this._currentNode.ChildNodes.Add(ast);
			}
		}

		// Token: 0x06000036 RID: 54 RVA: 0x00003568 File Offset: 0x00001768
		private void ParseEnd()
		{
			AST ast = null;
			AST ast2 = this.CreateElement(99);
			FglTokenNode fglTokenNode = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
			TokenType type = this.Predict().Type;
			if (type <= TokenType.FUNCTION_TOKEN)
			{
				if (type == TokenType.EOS_TOKEN)
				{
					return;
				}
				switch (type)
				{
				case TokenType.MAIN_TOKEN:
				case TokenType.DISPLAY_TOKEN:
				case TokenType.MENU_TOKEN:
				case TokenType.PROMPT_TOKEN:
				case TokenType.INPUT_TOKEN:
					break;
				case TokenType.TO_TOKEN:
					goto IL_00DF;
				case TokenType.DIALOG_TOKEN:
					ast = this.Find(this.Predict().Type);
					if (ast == null)
					{
						ast = this.Find(1);
						goto IL_00E1;
					}
					goto IL_00E1;
				default:
					if (type != TokenType.FUNCTION_TOKEN)
					{
						goto IL_00DF;
					}
					goto IL_00AB;
				}
			}
			else if (type != TokenType.CONSTRUCT_TOKEN)
			{
				switch (type)
				{
				case TokenType.FOR_TOEKN:
				case TokenType.WHILE_TOKEN:
				case TokenType.IF_TOKEN:
					ast2 = null;
					this.Next();
					goto IL_00E1;
				case TokenType.FOREACH_TOKEN:
				case TokenType.ATTRIBUTES_TOKEN:
					goto IL_00DF;
				default:
					if (type != TokenType.REPORT_TOKEN)
					{
						goto IL_00DF;
					}
					goto IL_00AB;
				}
			}
			ast = this.Find(this.Predict().Type);
			goto IL_00E1;
			IL_00AB:
			ast = this.Find(1);
			goto IL_00E1;
			IL_00DF:
			ast2 = null;
			IL_00E1:
			if (ast2 != null)
			{
				fglTokenNode.Children.Add(new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text)));
				ast2.TokenNode = fglTokenNode;
				if (ast != null)
				{
					ast.EndNode = ast2;
					this._currentNode = ast.ParentNode;
				}
			}
		}

		// Token: 0x06000037 RID: 55 RVA: 0x000036A4 File Offset: 0x000018A4
		private void ParseInteract()
		{
			AST ast = this._currentNode;
			if (ast.NodeKind == 3)
			{
				if (ast.EndNode != null)
				{
					ast = ast.ParentNode;
				}
				else if (ast.ChildNodes.Count == 0 && ast.TokenNode.Token.Type != TokenType.DIALOG_TOKEN)
				{
					ast = ast.ParentNode;
				}
				else if (ast.TokenNode.Token.Type == TokenType.DISPLAY_TOKEN || ast.TokenNode.Token.Type == TokenType.INPUT_TOKEN)
				{
					ast = ast.ParentNode;
				}
			}
			if (ast != null)
			{
				AST ast2 = this.CreateElement(3);
				FglTokenNode fglTokenNode = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
				TokenType type = fglTokenNode.Token.Type;
				switch (type)
				{
				case TokenType.DISPLAY_TOKEN:
				case TokenType.INPUT_TOKEN:
					if (this.Previous().Type == TokenType.CONTINUE_TOKEN)
					{
						ast2 = null;
					}
					else if (this.Predict().Type == TokenType.ARRAY_TOKEN)
					{
						FglTokenNode fglTokenNode2 = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
						if (this.Predict().Type == TokenType.UNKNOWN_TOKEN)
						{
							FglTokenNode fglTokenNode3 = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
							fglTokenNode.Children.Add(fglTokenNode2);
							fglTokenNode.Children.Add(fglTokenNode3);
							ast2.TokenNode = fglTokenNode;
						}
					}
					else if (this.Predict().Type == TokenType.UNKNOWN_TOKEN)
					{
						foreach (FglTokenNode fglTokenNode4 in this.GetParameters())
						{
							fglTokenNode.Children.Add(fglTokenNode4);
						}
						ast2.TokenNode = fglTokenNode;
					}
					else if (this.Predict().Type == TokenType.BY_TOKEN)
					{
						FglTokenNode fglTokenNode5 = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
						if (this.Predict().Type == TokenType.NAME_TOKEN)
						{
							FglTokenNode fglTokenNode6 = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
							fglTokenNode.Children.Add(fglTokenNode5);
							fglTokenNode.Children.Add(fglTokenNode6);
							while (this.Predict().Type == TokenType.UNKNOWN_TOKEN)
							{
								if (fglTokenNode.Children.Count > 0 && fglTokenNode.Children[fglTokenNode.Children.Count - 1].Token.Type == TokenType.UNKNOWN_TOKEN)
								{
									fglTokenNode.Children.Add(new FglTokenNode(new FglToken(TokenType.PEROID_TOKEN, ",")));
								}
								using (IEnumerator<FglTokenNode> enumerator2 = this.GetParameters().GetEnumerator())
								{
									while (enumerator2.MoveNext())
									{
										FglTokenNode fglTokenNode7 = enumerator2.Current;
										fglTokenNode.Children.Add(fglTokenNode7);
									}
									continue;
								}
								break;
							}
							ast2.TokenNode = fglTokenNode;
						}
					}
					else
					{
						ast2 = null;
					}
					break;
				case TokenType.MENU_TOKEN:
				case TokenType.DIALOG_TOKEN:
					ast2.TokenNode = fglTokenNode;
					break;
				case TokenType.PROMPT_TOKEN:
					ast2.TokenNode = fglTokenNode;
					break;
				case TokenType.TO_TOKEN:
					break;
				default:
					if (type == TokenType.CONSTRUCT_TOKEN)
					{
						if (this.Previous().Type == TokenType.CONTINUE_TOKEN)
						{
							ast2 = null;
						}
						else if (this.Predict().Type == TokenType.BY_TOKEN)
						{
							FglTokenNode fglTokenNode8 = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
							if (this.Predict().Type == TokenType.NAME_TOKEN)
							{
								FglTokenNode fglTokenNode9 = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
								if (this.Predict().Type == TokenType.UNKNOWN_TOKEN)
								{
									List<FglTokenNode> list = this.GetParameters().ToList<FglTokenNode>();
									if (list.Count > 0)
									{
										fglTokenNode.Children.Add(fglTokenNode8);
										fglTokenNode.Children.Add(fglTokenNode9);
										foreach (FglTokenNode fglTokenNode10 in list)
										{
											fglTokenNode.Children.Add(fglTokenNode10);
										}
										ast2.TokenNode = fglTokenNode;
									}
								}
							}
						}
						else if (this.Predict().Type == TokenType.UNKNOWN_TOKEN)
						{
							foreach (FglTokenNode fglTokenNode11 in this.GetParameters())
							{
								fglTokenNode.Children.Add(fglTokenNode11);
							}
							if (fglTokenNode.Count > 0)
							{
								ast2.TokenNode = fglTokenNode;
							}
						}
						else
						{
							ast2 = null;
						}
					}
					break;
				}
				if (ast2 != null)
				{
					ast2.ParentNode = ast;
					ast.ChildNodes.Add(ast2);
					this._currentNode = ast2;
				}
			}
		}

		// Token: 0x06000038 RID: 56 RVA: 0x00003CEC File Offset: 0x00001EEC
		private IEnumerable<FglTokenNode> GetParameters()
		{
			bool isEnd = false;
			while (!isEnd)
			{
				TokenType type = this.Predict().Type;
				switch (type)
				{
				case TokenType.PEROID_TOKEN:
					yield return new FglTokenNode(this.Next());
					break;
				case TokenType.EOS_TOKEN:
					isEnd = true;
					break;
				default:
					if (type == TokenType.UNKNOWN_TOKEN)
					{
						yield return new FglTokenNode(this.Next());
					}
					else
					{
						isEnd = true;
					}
					break;
				}
			}
			yield break;
		}

		// Token: 0x06000039 RID: 57 RVA: 0x00003D0C File Offset: 0x00001F0C
		private void ParseFunction()
		{
			FunctionAST functionAST = this.CreateFunctionElement(1);
			if (functionAST == null)
			{
				return;
			}
			functionAST.Scope = this.Previous();
			if (this.Predict().Type == TokenType.FUNCTION_TOKEN)
			{
				this.Next();
				if (this.Predict().Type == TokenType.UNKNOWN_TOKEN)
				{
					FglTokenNode fglTokenNode = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
					if (this.Predict().Type == TokenType.LEFTPAREN_TOKEN)
					{
						this.Next();
						functionAST.TokenNode = fglTokenNode;
					}
					else
					{
						functionAST = null;
					}
				}
				else
				{
					functionAST = null;
				}
			}
			if (functionAST != null)
			{
				this._currentNode.ChildNodes.Add(functionAST);
				functionAST.ParentNode = this._currentNode;
				this._currentNode = functionAST;
			}
		}

		// Token: 0x0600003A RID: 58 RVA: 0x00003DC4 File Offset: 0x00001FC4
		private void ParseReportDialogFunction()
		{
			FunctionAST functionAST = this.CreateFunctionElement(1);
			if (functionAST == null)
			{
				return;
			}
			TokenType type = this.Previous().Type;
			if (type == TokenType.PRIVATE_TOKEN)
			{
				functionAST.Scope = new FglToken(this.Previous().Type, this.Previous().Text);
			}
			else
			{
				functionAST.Scope = new FglToken(TokenType.PUBLIC_TOKEN, "PUBLIC");
			}
			if (this.Predict().Type == TokenType.REPORT_TOKEN || this.Predict().Type == TokenType.DIALOG_TOKEN)
			{
				this.Next();
				if (this.Predict().Type == TokenType.UNKNOWN_TOKEN)
				{
					FglTokenNode fglTokenNode = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
					if (this.Predict().Type == TokenType.LEFTPAREN_TOKEN)
					{
						this.Next();
						functionAST.TokenNode = fglTokenNode;
					}
					else
					{
						functionAST = null;
					}
				}
				else
				{
					functionAST = null;
				}
			}
			if (functionAST != null)
			{
				this._currentNode.ChildNodes.Add(functionAST);
				functionAST.ParentNode = this._currentNode;
				this._currentNode = functionAST;
			}
		}

		// Token: 0x0600003B RID: 59 RVA: 0x00003EC4 File Offset: 0x000020C4
		private void ParseMainFunction()
		{
			if (this._currentNode.NodeKind != 0)
			{
				throw new FormatException("parent is not root");
			}
			FunctionAST functionAST = this.CreateFunctionElement(1);
			if (functionAST == null)
			{
				return;
			}
			functionAST.Scope = new FglToken(TokenType.PUBLIC_TOKEN, "");
			functionAST.TokenNode = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
			this._currentNode.ChildNodes.Add(functionAST);
			functionAST.ParentNode = this._currentNode;
			this._currentNode = functionAST;
		}

		// Token: 0x0600003C RID: 60 RVA: 0x00003F54 File Offset: 0x00002154
		private void ParseAfterBlock()
		{
			this.SetCurrentNode();
			AST ast = this.CreateElement(4);
			FglTokenNode fglTokenNode = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
			TokenType type = this.Predict().Type;
			if (type <= TokenType.DISPLAY_TOKEN)
			{
				if (type == TokenType.EOS_TOKEN)
				{
					return;
				}
				if (type != TokenType.DISPLAY_TOKEN)
				{
					goto IL_0126;
				}
			}
			else
			{
				switch (type)
				{
				case TokenType.DIALOG_TOKEN:
				case TokenType.INPUT_TOKEN:
					break;
				default:
					switch (type)
					{
					case TokenType.FIELD_TOKEN:
						fglTokenNode.Children.Add(new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text)));
						if (this.Predict().Type == TokenType.UNKNOWN_TOKEN)
						{
							fglTokenNode.Children.Add(new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text)));
							goto IL_0128;
						}
						goto IL_0128;
					case TokenType.FUNCTION_TOKEN:
					case TokenType.WITHOUT_TOKEN:
					case TokenType.DEFAULTS_TOKEN:
					case TokenType.FROM_TOKEN:
						goto IL_0126;
					case TokenType.CONSTRUCT_TOKEN:
					case TokenType.ROW_TOKEN:
					case TokenType.INSERT_TOKEN:
					case TokenType.DELETE_TOKEN:
						break;
					default:
						goto IL_0126;
					}
					break;
				}
			}
			fglTokenNode.Children.Add(new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text)));
			goto IL_0128;
			IL_0126:
			ast = null;
			IL_0128:
			if (ast != null)
			{
				ast.TokenNode = fglTokenNode;
				this.SetCurrentNode();
				this._currentNode.ChildNodes.Add(ast);
				this._currentNode = ast;
			}
		}

		// Token: 0x0600003D RID: 61 RVA: 0x000040B4 File Offset: 0x000022B4
		private void ParseBeforeBlock()
		{
			this.SetCurrentNode();
			AST ast = this.CreateElement(4);
			FglTokenNode fglTokenNode = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
			TokenType type = this.Predict().Type;
			if (type != TokenType.EOS_TOKEN)
			{
				switch (type)
				{
				case TokenType.DISPLAY_TOKEN:
				case TokenType.DIALOG_TOKEN:
				case TokenType.INPUT_TOKEN:
					break;
				case TokenType.MENU_TOKEN:
					fglTokenNode.Children.Add(new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text)));
					goto IL_0189;
				case TokenType.PROMPT_TOKEN:
				case TokenType.TO_TOKEN:
					goto IL_0187;
				default:
					switch (type)
					{
					case TokenType.FIELD_TOKEN:
						fglTokenNode.Children.Add(new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text)));
						if (this.Predict().Type == TokenType.UNKNOWN_TOKEN)
						{
							fglTokenNode.Children.Add(new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text)));
							goto IL_0189;
						}
						goto IL_0189;
					case TokenType.FUNCTION_TOKEN:
					case TokenType.WITHOUT_TOKEN:
					case TokenType.DEFAULTS_TOKEN:
					case TokenType.FROM_TOKEN:
						goto IL_0187;
					case TokenType.CONSTRUCT_TOKEN:
						break;
					case TokenType.ROW_TOKEN:
					case TokenType.INSERT_TOKEN:
					case TokenType.DELETE_TOKEN:
						fglTokenNode.Children.Add(new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text)));
						goto IL_0189;
					default:
						goto IL_0187;
					}
					break;
				}
				fglTokenNode.Children.Add(new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text)));
				goto IL_0189;
				IL_0187:
				ast = null;
				IL_0189:
				if (ast != null)
				{
					ast.TokenNode = fglTokenNode;
					this.SetCurrentNode();
					this._currentNode.ChildNodes.Add(ast);
					this._currentNode = ast;
				}
				return;
			}
		}

		// Token: 0x0600003E RID: 62 RVA: 0x00004274 File Offset: 0x00002474
		private void ParseOnBlock()
		{
			this.SetCurrentNode();
			AST ast = this.CreateElement(4);
			FglTokenNode fglTokenNode = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
			TokenType type = this.Predict().Type;
			if (type != TokenType.CHANGE_TOKEN && type != TokenType.ACTION_TOKEN)
			{
				if (type != TokenType.ROW_TOKEN)
				{
					ast = null;
				}
				else
				{
					FglTokenNode fglTokenNode2 = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
					if (this.Predict().Type == TokenType.CHANGE_TOKEN)
					{
						FglTokenNode fglTokenNode3 = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
						fglTokenNode.Children.Add(fglTokenNode2);
						fglTokenNode.Children.Add(fglTokenNode3);
					}
					ast.TokenNode = fglTokenNode;
				}
			}
			else
			{
				FglTokenNode fglTokenNode4 = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
				FglTokenNode fglTokenNode5 = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
				fglTokenNode.Children.Add(fglTokenNode4);
				fglTokenNode.Children.Add(fglTokenNode5);
				if (this.Predict().Type == TokenType.INFIELD_TOKEN)
				{
					FglTokenNode fglTokenNode6 = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
					if (this.Predict().Type == TokenType.UNKNOWN_TOKEN)
					{
						FglTokenNode fglTokenNode7 = new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
						fglTokenNode.Children.Add(fglTokenNode6);
						fglTokenNode.Children.Add(fglTokenNode7);
					}
				}
				ast.TokenNode = fglTokenNode;
			}
			if (ast != null)
			{
				this._currentNode.ChildNodes.Add(ast);
				ast.ParentNode = this._currentNode;
				this._currentNode = ast;
			}
		}

		// Token: 0x0600003F RID: 63 RVA: 0x00004458 File Offset: 0x00002658
		private void SetCurrentNode()
		{
			int nodeKind = this._currentNode.NodeKind;
			switch (nodeKind)
			{
			case 0:
			case 2:
				return;
			case 1:
				if (this._currentNode.EndNode != null)
				{
					this.SetCurrentNode();
					return;
				}
				return;
			case 3:
				break;
			case 4:
				if (this._currentNode.ParentNode.NodeKind == 3)
				{
					this._currentNode = this._currentNode.ParentNode;
					this.SetCurrentNode();
					return;
				}
				return;
			default:
				if (nodeKind != 99)
				{
					return;
				}
				break;
			}
			if (this._currentNode.ParentNode.TokenNode.Token.Type == TokenType.DIALOG_TOKEN)
			{
				return;
			}
			if (this._currentNode.ParentNode.NodeKind == 1)
			{
				return;
			}
			TokenType type = this._currentNode.ParentNode.TokenNode.Token.Type;
			if (type != TokenType.MAIN_TOKEN && type != TokenType.FUNCTION_TOKEN && (this._currentNode.ParentNode.NodeKind == 4 || this._currentNode.ParentNode.NodeKind == 3))
			{
				if (this._currentNode.ParentNode.TokenNode.Token.Type == TokenType.ON_TOKEN)
				{
					return;
				}
				this._currentNode = this._currentNode.ParentNode;
				this.SetCurrentNode();
			}
		}

		// Token: 0x06000040 RID: 64 RVA: 0x0000458C File Offset: 0x0000278C
		private void ParseOthersBlock()
		{
			this.CreateElement(4);
			switch (this.Predict().Type)
			{
			case TokenType.EXIT_TOKEN:
			case TokenType.CONTINUE_TOKEN:
			{
				this.Next();
				TokenType type = this.Predict().Type;
				switch (type)
				{
				case TokenType.DISPLAY_TOKEN:
				case TokenType.MENU_TOKEN:
				case TokenType.DIALOG_TOKEN:
					break;
				case TokenType.PROMPT_TOKEN:
				case TokenType.TO_TOKEN:
					return;
				default:
					if (type != TokenType.CONSTRUCT_TOKEN)
					{
						switch (type)
						{
						case TokenType.FOR_TOEKN:
						case TokenType.FOREACH_TOKEN:
						case TokenType.WHILE_TOKEN:
						case TokenType.CASE_TOKEN:
						case TokenType.REPORT_TOKEN:
							break;
						case TokenType.ATTRIBUTES_TOKEN:
						case TokenType.IF_TOKEN:
						case TokenType.START_TOKEN:
							return;
						default:
							return;
						}
					}
					break;
				}
				this.Next();
				return;
			}
			case TokenType.ACCEPT_TOKEN:
			{
				this.Next();
				TokenType type2 = this.Predict().Type;
				if (type2 != TokenType.DISPLAY_TOKEN)
				{
					switch (type2)
					{
					case TokenType.DIALOG_TOKEN:
					case TokenType.INPUT_TOKEN:
						break;
					default:
						if (type2 != TokenType.CONSTRUCT_TOKEN)
						{
							return;
						}
						break;
					}
				}
				this.Next();
				break;
			}
			default:
				return;
			}
		}

		// Token: 0x06000041 RID: 65 RVA: 0x00004660 File Offset: 0x00002860
		private AST Find(TokenType targetType)
		{
			AST ast = this._currentNode;
			while (ast.NodeKind != 0)
			{
				if (ast.TokenNode != null && ast.TokenNode.Token.Type == targetType && ((ast.NodeKind == 3 && ast.ChildNodes.Count > 0 && ast.EndNode == null) || (ast.NodeKind == 1 && ast.EndNode == null)))
				{
					return ast;
				}
				ast = ast.ParentNode;
			}
			ast = this._currentNode;
			return null;
		}

		// Token: 0x06000042 RID: 66 RVA: 0x000046DC File Offset: 0x000028DC
		private AST Find(int targetNodeKind)
		{
			AST ast = this._currentNode;
			for (;;)
			{
				if (ast != null)
				{
					if (ast.NodeKind == targetNodeKind)
					{
						return ast;
					}
					if (ast.ParentNode == null)
					{
						break;
					}
					ast = ast.ParentNode;
				}
				else
				{
					if (ast == null)
					{
						return ast;
					}
					if (ast.NodeKind == 0)
					{
						goto Block_5;
					}
				}
			}
			return ast;
			Block_5:
			return null;
		}

		// Token: 0x06000043 RID: 67 RVA: 0x00004720 File Offset: 0x00002920
		private void ParseWhenever()
		{
			new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
			if (this.Predict().Type == TokenType.ANY_TOKEN)
			{
				this.Next();
				switch (this.Predict().Type)
				{
				case TokenType.ERROR_TOKEN:
				case TokenType.SQLERROR_TOKEN:
				case TokenType.WARNING_TOKEN:
				{
					this.Next();
					TokenType type = this.Predict().Type;
					switch (type)
					{
					case TokenType.CONTINUE_TOKEN:
					case TokenType.CALL_TOKEN:
						break;
					case TokenType.ATTRIBUTE_TOKEN:
						return;
					default:
						switch (type)
						{
						case TokenType.STOP_TOKEN:
						case TokenType.RAISE_TOKEN:
						case TokenType.GOTO_TOKEN:
							break;
						default:
							return;
						}
						break;
					}
					this.Next();
					return;
				}
				case TokenType.NOT_TOKEN:
					this.Next();
					if (this.Predict().Type == TokenType.FOUND_TOKEN)
					{
						this.Next();
						TokenType type2 = this.Predict().Type;
						switch (type2)
						{
						case TokenType.CONTINUE_TOKEN:
						case TokenType.CALL_TOKEN:
							break;
						case TokenType.ATTRIBUTE_TOKEN:
							return;
						default:
							switch (type2)
							{
							case TokenType.STOP_TOKEN:
							case TokenType.RAISE_TOKEN:
							case TokenType.GOTO_TOKEN:
								break;
							default:
								return;
							}
							break;
						}
						this.Next();
						return;
					}
					break;
				case TokenType.FOUND_TOKEN:
					break;
				default:
					return;
				}
			}
			else
			{
				switch (this.Predict().Type)
				{
				case TokenType.ERROR_TOKEN:
				case TokenType.SQLERROR_TOKEN:
				case TokenType.WARNING_TOKEN:
				{
					this.Next();
					TokenType type3 = this.Predict().Type;
					switch (type3)
					{
					case TokenType.CONTINUE_TOKEN:
					case TokenType.CALL_TOKEN:
						break;
					case TokenType.ATTRIBUTE_TOKEN:
						return;
					default:
						switch (type3)
						{
						case TokenType.STOP_TOKEN:
						case TokenType.RAISE_TOKEN:
						case TokenType.GOTO_TOKEN:
							break;
						default:
							return;
						}
						break;
					}
					this.Next();
					return;
				}
				case TokenType.NOT_TOKEN:
					this.Next();
					if (this.Predict().Type == TokenType.FOUND_TOKEN)
					{
						this.Next();
						TokenType type4 = this.Predict().Type;
						switch (type4)
						{
						case TokenType.CONTINUE_TOKEN:
						case TokenType.CALL_TOKEN:
							break;
						case TokenType.ATTRIBUTE_TOKEN:
							return;
						default:
							switch (type4)
							{
							case TokenType.STOP_TOKEN:
							case TokenType.RAISE_TOKEN:
							case TokenType.GOTO_TOKEN:
								break;
							default:
								return;
							}
							break;
						}
						this.Next();
					}
					break;
				case TokenType.FOUND_TOKEN:
					break;
				default:
					return;
				}
			}
		}

		// Token: 0x06000044 RID: 68 RVA: 0x00004908 File Offset: 0x00002B08
		private void ParseErrorCheck()
		{
			new FglTokenNode(new FglToken(this.Predict().Type, this.Next().Text));
			TokenType type = this.Predict().Type;
			if (type != TokenType.END_TOKEN)
			{
			}
		}

		// Token: 0x04000021 RID: 33
		private FglScanner _scanner;

		// Token: 0x04000022 RID: 34
		private FglToken _currentToken;

		// Token: 0x04000023 RID: 35
		private FglToken _predictToken;

		// Token: 0x04000024 RID: 36
		private bool _used;

		// Token: 0x04000025 RID: 37
		private AST _root;

		// Token: 0x04000026 RID: 38
		private FglToken _previous;

		// Token: 0x04000027 RID: 39
		private bool isCancelled;

		// Token: 0x04000028 RID: 40
		private AST _currentNode;
	}
}
