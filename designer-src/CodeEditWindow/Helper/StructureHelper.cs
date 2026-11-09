using System;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Threading;
using ICSharpCode.AvalonEdit.Document;
using SpecDesigner.CodeEditWindow.View;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000005 RID: 5
	public class StructureHelper : IDisposable
	{
		// Token: 0x0600000D RID: 13 RVA: 0x0000215C File Offset: 0x0000035C
		public StructureHelper(PackageKey key, CodeTextEditor editor)
		{
			this._programKey = key;
			this._editor = editor;
			this._editor.Loaded += this._editor_Loaded;
		}

		// Token: 0x0600000E RID: 14 RVA: 0x000021BD File Offset: 0x000003BD
		private void _editor_Loaded(object sender, RoutedEventArgs e)
		{
			this._editor.Loaded -= this._editor_Loaded;
			this._editor.TextArea.Caret.PositionChanged += this.Caret_PositionChanged;
		}

		// Token: 0x0600000F RID: 15 RVA: 0x000023F1 File Offset: 0x000005F1
		private void Caret_PositionChanged(object sender, EventArgs e)
		{
			this.triggerTimes++;
			Dispatcher.CurrentDispatcher.BeginInvoke(new Action(delegate
			{
				if (this.triggerTimes > 1)
				{
					this.triggerTimes--;
					return;
				}
				ProgramInformation programInfo = ResourceController.GetInstance().GetProgramInfo(this._programKey);
				if (programInfo == null)
				{
					return;
				}
				bool flag = false;
				int lineNumber = this._editor.Document.GetLineByOffset(this._editor.CaretOffset).LineNumber;
				SegmentObject segment = this._editor.Document.SectionProvider.Find(lineNumber);
				if (segment != null)
				{
					AddPointModel addPointModel = programInfo.AddPoints.Where<AddPointModel>((AddPointModel ap) => (ap.Status & Status.DELETE) != Status.DELETE && ap.ID == segment.ID).FirstOrDefault<AddPointModel>();
					if (addPointModel != null && addPointModel.IsSelfDefinition)
					{
						TreeItem treeItem = programInfo.Find(addPointModel.FunctionNameWithoutParameter).FirstOrDefault<TreeItem>();
						if (treeItem != null)
						{
							treeItem.IsSelected = true;
							flag = true;
						}
					}
				}
				if (!flag)
				{
					int endOffset = this._editor.Document.GetLineByOffset(this._editor.CaretOffset).EndOffset;
					string text = Regex.Replace(this._editor.Text.Substring(0, endOffset), this.areaMarkedPattern, "");
					Match match = null;
					if (Regex.IsMatch(text, this.funcPattern, this.options))
					{
						match = Regex.Match(text, this.funcPattern, this.options);
					}
					else if (Regex.IsMatch(text, this.mainPattern, this.options))
					{
						match = Regex.Match(text, this.mainPattern, this.options);
					}
					if (match != null)
					{
						TreeItem treeItem2 = programInfo.Find(match.Groups["name"].Value).ElementAtOrDefault<TreeItem>(0);
						if (treeItem2 != null)
						{
							treeItem2.IsSelected = true;
						}
					}
				}
				this.triggerTimes--;
			}), DispatcherPriority.ApplicationIdle, new object[0]);
		}

		// Token: 0x06000010 RID: 16 RVA: 0x0000241F File Offset: 0x0000061F
		public void Dispose()
		{
			this._editor.TextArea.Caret.PositionChanged -= this.Caret_PositionChanged;
		}

		// Token: 0x04000004 RID: 4
		private CodeTextEditor _editor;

		// Token: 0x04000005 RID: 5
		private PackageKey _programKey;

		// Token: 0x04000006 RID: 6
		private RegexOptions options = RegexOptions.Multiline | RegexOptions.RightToLeft;

		// Token: 0x04000007 RID: 7
		private readonly string funcPattern = "^(?:PUBLIC\\s*|PRIVATE\\s*|)(?:FUNCTION|REPORT|DIALOG)\\s+(?<name>\\w+)(?:\\([^\\)]*\\))";

		// Token: 0x04000008 RID: 8
		private readonly string mainPattern = "^(?<name>MAIN)";

		// Token: 0x04000009 RID: 9
		private readonly string areaMarkedPattern = "(?:{[^}]*\\})";

		// Token: 0x0400000A RID: 10
		private int triggerTimes;
	}
}
