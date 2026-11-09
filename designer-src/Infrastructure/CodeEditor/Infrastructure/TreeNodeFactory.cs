using System;
using System.Collections.Generic;
using CodeEditor.FglAnalysis;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace CodeEditor.Infrastructure
{
	// Token: 0x0200003F RID: 63
	public class TreeNodeFactory
	{
		// Token: 0x06000173 RID: 371 RVA: 0x000070BC File Offset: 0x000052BC
		public static TreeItem Parse(AddPointModel addPointModel)
		{
			return new TreeItem(string.Format("{0} {1} {2}", addPointModel.Scope, addPointModel.Type, addPointModel.FunctionName))
			{
				ReferenceName = addPointModel.Name,
				Description = addPointModel.Description,
				Scope = addPointModel.Scope,
				ProgramKey = addPointModel.ProgramKey,
				SortIndex = addPointModel.SortIndex,
				Type = addPointModel.Type
			};
		}

		// Token: 0x06000174 RID: 372 RVA: 0x00007140 File Offset: 0x00005340
		public static void CloneFrom(ref TreeItem treeItem, AddPointModel model)
		{
			treeItem.ReferenceName = model.Name;
			treeItem.Description = model.Description;
			treeItem.Scope = model.Scope;
			treeItem.ProgramKey = model.ProgramKey;
			treeItem.SortIndex = model.SortIndex;
			treeItem.Type = model.Type;
		}

		// Token: 0x06000175 RID: 373 RVA: 0x0000719C File Offset: 0x0000539C
		public static List<TreeItem> Parse(string content, PackageKey key)
		{
			FglScanner fglScanner = new FglScanner(new FglReader(content));
			FglParser fglParser = new FglParser(fglScanner);
			AST ast = fglParser.Parse();
			List<TreeItem> list = new List<TreeItem>();
			foreach (AST ast2 in ast.ChildNodes)
			{
				list.Add(TreeNodeFactory.GenerateTreeItemFromAST(key, ast2));
			}
			return list;
		}

		// Token: 0x06000176 RID: 374 RVA: 0x00007218 File Offset: 0x00005418
		private static TreeItem GenerateTreeItemFromAST(PackageKey key, AST source)
		{
			TreeItem treeItem = new TreeItem(source.TokenNode.ToString())
			{
				ProgramKey = key
			};
			if (source.NodeKind == 1)
			{
				FunctionAST functionAST = source as FunctionAST;
				if (functionAST != null && functionAST.EndNode != null)
				{
					TokenType type = functionAST.EndNode.TokenNode.Children[0].Token.Type;
					if (type != TokenType.DIALOG_TOKEN)
					{
						if (type != TokenType.FUNCTION_TOKEN)
						{
							if (type == TokenType.REPORT_TOKEN)
							{
								treeItem.Type = DefinitionType.REPORT;
							}
						}
						else
						{
							treeItem.Type = DefinitionType.FUNCTION;
						}
					}
					else
					{
						treeItem.Type = DefinitionType.DIALOG;
					}
					treeItem.Scope = ((functionAST.Scope.Type == TokenType.PRIVATE_TOKEN) ? Scope.PRIVATE : Scope.PUBLIC);
				}
			}
			if (source.ChildNodes.Count > 0)
			{
				foreach (AST ast in source.ChildNodes)
				{
					if (ast.NodeKind != 3 || ast.EndNode != null)
					{
						TokenType type2 = ast.TokenNode.Token.Type;
						if (type2 != TokenType.MENU_TOKEN)
						{
							treeItem.AddNode(TreeNodeFactory.GenerateTreeItemFromAST(key, ast));
						}
					}
				}
			}
			return treeItem;
		}
	}
}
