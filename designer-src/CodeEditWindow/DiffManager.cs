using System;
using System.Collections;
using System.Collections.Generic;
using System.Text;
using System.Text.RegularExpressions;
using DifferenceEngine;
using ICSharpCode.AvalonEdit.Highlighting;
using SpecDesigner.CodeEditWindow.View;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;
using SpecDesignerPreference;

namespace SpecDesigner.CodeEditWindow
{
	// Token: 0x02000027 RID: 39
	public class DiffManager : IDisposable
	{
		// Token: 0x17000029 RID: 41
		// (get) Token: 0x06000154 RID: 340 RVA: 0x0000C60E File Offset: 0x0000A80E
		// (set) Token: 0x06000155 RID: 341 RVA: 0x0000C616 File Offset: 0x0000A816
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x1700002A RID: 42
		// (get) Token: 0x06000156 RID: 342 RVA: 0x0000C61F File Offset: 0x0000A81F
		// (set) Token: 0x06000157 RID: 343 RVA: 0x0000C627 File Offset: 0x0000A827
		public DiffTextViewer SourceEditor
		{
			get
			{
				return this._sourceEditor;
			}
			set
			{
				this._sourceEditor = value;
			}
		}

		// Token: 0x1700002B RID: 43
		// (get) Token: 0x06000158 RID: 344 RVA: 0x0000C630 File Offset: 0x0000A830
		// (set) Token: 0x06000159 RID: 345 RVA: 0x0000C638 File Offset: 0x0000A838
		public DiffTextViewer TargetEditor
		{
			get
			{
				return this._targeEditor;
			}
			set
			{
				this._targeEditor = value;
			}
		}

		// Token: 0x0600015A RID: 346 RVA: 0x0000C641 File Offset: 0x0000A841
		private DiffManager()
		{
		}

		// Token: 0x0600015B RID: 347 RVA: 0x0000C64C File Offset: 0x0000A84C
		private DiffManager(IComparisonWindow view)
		{
			this.View = view;
			this.View.SourceViewer.TextArea.TextView.VisualLinesChanged += this.main_VisualLinesChanged;
			this.View.TargetViewer.TextArea.TextView.VisualLinesChanged += this.diff_VisualLinesChanged;
		}

		// Token: 0x0600015C RID: 348 RVA: 0x0000C6B4 File Offset: 0x0000A8B4
		public DiffManager(DiffTextViewer SourceViewer, string source, DiffTextViewer TargetViewer, string target, bool withColor, PackageKey key)
		{
			SourceViewer.TextArea.TextView.VisualLinesChanged += this.mainDiffBaseOnStandard_VisualLinesChanged;
			TargetViewer.TextArea.TextView.VisualLinesChanged += this.diffDiffBaseOnStandard_VisualLinesChanged;
			this.ProgramKey = key;
			this._sourceEditor = SourceViewer;
			this._targeEditor = TargetViewer;
			this.MoveBetweenDiff = true;
			ArrayList arrayList = DiffManager.SimpleDiff(source, target);
			this.DiffContent(SourceViewer, source, TargetViewer, target, arrayList, withColor);
			this.ReturnSourceViewer = SourceViewer;
			this.ReturnTargetViewer = TargetViewer;
		}

		// Token: 0x0600015D RID: 349 RVA: 0x0000C744 File Offset: 0x0000A944
		public DiffManager(IComparisonWindow view, string source, string target)
			: this(view)
		{
			ArrayList arrayList = DiffManager.SimpleDiff(source, target);
			this.DiffContent(view.SourceViewer, source, view.TargetViewer, target, arrayList, false);
		}

		// Token: 0x1700002C RID: 44
		// (get) Token: 0x0600015E RID: 350 RVA: 0x0000C776 File Offset: 0x0000A976
		// (set) Token: 0x0600015F RID: 351 RVA: 0x0000C77E File Offset: 0x0000A97E
		public bool MoveBetweenDiff { get; set; }

		// Token: 0x1700002D RID: 45
		// (get) Token: 0x06000160 RID: 352 RVA: 0x0000C787 File Offset: 0x0000A987
		// (set) Token: 0x06000161 RID: 353 RVA: 0x0000C78F File Offset: 0x0000A98F
		public DiffTextViewer ReturnSourceViewer { get; set; }

		// Token: 0x1700002E RID: 46
		// (get) Token: 0x06000162 RID: 354 RVA: 0x0000C798 File Offset: 0x0000A998
		// (set) Token: 0x06000163 RID: 355 RVA: 0x0000C7A0 File Offset: 0x0000A9A0
		public DiffTextViewer ReturnTargetViewer { get; set; }

		// Token: 0x06000164 RID: 356 RVA: 0x0000C7AC File Offset: 0x0000A9AC
		private void main_VisualLinesChanged(object sender, EventArgs e)
		{
			double y = this.View.SourceViewer.TextArea.TextView.ScrollOffset.Y;
			double y2 = this.View.TargetViewer.TextArea.TextView.ScrollOffset.Y;
			if (Math.Abs(y2 - y) > this.View.TargetViewer.TextArea.TextView.DefaultLineHeight)
			{
				this.View.TargetViewer.TextArea.ScrollToVerticalPosition(y);
			}
			double x = this.View.SourceViewer.TextArea.TextView.ScrollOffset.X;
			double x2 = this.View.TargetViewer.TextArea.TextView.ScrollOffset.X;
			this.View.TargetViewer.TextArea.ScrollToHorizontalPosition(x);
		}

		// Token: 0x06000165 RID: 357 RVA: 0x0000C898 File Offset: 0x0000AA98
		private void diff_VisualLinesChanged(object sender, EventArgs e)
		{
			double y = this.View.SourceViewer.TextArea.TextView.ScrollOffset.Y;
			double y2 = this.View.TargetViewer.TextArea.TextView.ScrollOffset.Y;
			if (Math.Abs(y2 - y) > this.View.SourceViewer.TextArea.TextView.DefaultLineHeight)
			{
				this.View.SourceViewer.TextArea.ScrollToVerticalPosition(y2);
			}
			double x = this.View.SourceViewer.TextArea.TextView.ScrollOffset.X;
			double x2 = this.View.TargetViewer.TextArea.TextView.ScrollOffset.X;
			this.View.TargetViewer.TextArea.ScrollToHorizontalPosition(x2);
		}

		// Token: 0x06000166 RID: 358 RVA: 0x0000C984 File Offset: 0x0000AB84
		private void mainDiffBaseOnStandard_VisualLinesChanged(object sender, EventArgs e)
		{
			double y = this.SourceEditor.TextArea.TextView.ScrollOffset.Y;
			double y2 = this.TargetEditor.TextArea.TextView.ScrollOffset.Y;
			if (Math.Abs(y2 - y) > this.TargetEditor.TextArea.TextView.DefaultLineHeight)
			{
				this.TargetEditor.TextArea.ScrollToVerticalPosition(y);
			}
			double x = this.SourceEditor.TextArea.TextView.ScrollOffset.X;
			double x2 = this.TargetEditor.TextArea.TextView.ScrollOffset.X;
			this.TargetEditor.TextArea.ScrollToHorizontalPosition(x);
		}

		// Token: 0x06000167 RID: 359 RVA: 0x0000CA50 File Offset: 0x0000AC50
		private void diffDiffBaseOnStandard_VisualLinesChanged(object sender, EventArgs e)
		{
			double y = this.SourceEditor.TextArea.TextView.ScrollOffset.Y;
			double y2 = this.TargetEditor.TextArea.TextView.ScrollOffset.Y;
			if (Math.Abs(y2 - y) > this.SourceEditor.TextArea.TextView.DefaultLineHeight)
			{
				this.SourceEditor.TextArea.ScrollToVerticalPosition(y2);
			}
			double x = this.SourceEditor.TextArea.TextView.ScrollOffset.X;
			double x2 = this.TargetEditor.TextArea.TextView.ScrollOffset.X;
			this.TargetEditor.TextArea.ScrollToHorizontalPosition(x2);
		}

		// Token: 0x06000168 RID: 360 RVA: 0x0000CB1C File Offset: 0x0000AD1C
		public void Dispose()
		{
			this.View.SourceViewer.TextArea.TextView.VisualLinesChanged -= this.main_VisualLinesChanged;
			this.View.TargetViewer.TextArea.TextView.VisualLinesChanged -= this.diff_VisualLinesChanged;
		}

		// Token: 0x06000169 RID: 361 RVA: 0x0000CB78 File Offset: 0x0000AD78
		public static ArrayList SimpleDiff(string context1, string context2)
		{
			DiffEngine diffEngine = new DiffEngine();
			diffEngine.ProcessDiff(new DiffList_TextFile(context1), new DiffList_TextFile(context2), DiffEngineLevel.FastImperfect);
			return diffEngine.DiffReport();
		}

		// Token: 0x0600016A RID: 362 RVA: 0x0000CBA8 File Offset: 0x0000ADA8
		public void DiffContent(DiffTextViewer sourceViewer, string context1, DiffTextViewer targetViewer, string context2, ArrayList rep, bool withColor)
		{
			sourceViewer.ClearRenderRegion();
			targetViewer.ClearRenderRegion();
			DiffList_TextFile diffList_TextFile = new DiffList_TextFile(context1);
			DiffList_TextFile diffList_TextFile2 = new DiffList_TextFile(context2);
			StringBuilder stringBuilder = new StringBuilder();
			StringBuilder stringBuilder2 = new StringBuilder();
			int num = 0;
			int num2 = 0;
			int num3 = 0;
			int num4 = 0;
			List<DiffColor> list = new List<DiffColor>();
			foreach (object obj in rep)
			{
				DiffResultSpan diffResultSpan = (DiffResultSpan)obj;
				switch (diffResultSpan.Status)
				{
				case DiffResultSpanStatus.NoChange:
				{
					for (int i = 0; i < diffResultSpan.Length; i++)
					{
						if (num < diffList_TextFile.Count() - 1)
						{
							stringBuilder.Append(((TextLine)diffList_TextFile.GetByIndex(num)).Line + "\r\n");
							stringBuilder2.Append(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line + "\r\n");
						}
						else
						{
							stringBuilder.Append(((TextLine)diffList_TextFile.GetByIndex(num)).Line);
							stringBuilder2.Append(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line);
						}
						num++;
						num2++;
					}
					break;
				}
				case DiffResultSpanStatus.Replace:
				{
					bool flag = false;
					for (int j = 0; j < diffResultSpan.Length; j++)
					{
						if (num < diffList_TextFile.Count() - 1 || num2 < diffList_TextFile2.Count() - 1)
						{
							stringBuilder.Append(((TextLine)diffList_TextFile.GetByIndex(num)).Line + "\r\n");
							stringBuilder2.Append(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line + "\r\n");
						}
						else
						{
							stringBuilder.Append(((TextLine)diffList_TextFile.GetByIndex(num)).Line);
							stringBuilder2.Append(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line);
						}
						if (num + 1 <= diffList_TextFile.Count() && num2 + 1 <= diffList_TextFile2.Count())
						{
							if ((Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*#") && Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "^\\s*#")) || (Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*#") && string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""))) || (string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "^\\s*#")))
							{
								if (PreferenceManager.Current.Settings.DiffComment)
								{
									if (!string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && !string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
									{
										if (!PreferenceManager.Current.Settings.SimplifyColor)
										{
											sourceViewer.AddRenderRegion(num + 1 + num3, DiffResultSpanStatus.Replace);
											targetViewer.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.Replace);
										}
										flag = true;
									}
									else if (!string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
									{
										if (!PreferenceManager.Current.Settings.SimplifyColor)
										{
											targetViewer.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.DeleteSource);
										}
										flag = true;
									}
									else if (string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && !string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
									{
										if (!PreferenceManager.Current.Settings.SimplifyColor)
										{
											sourceViewer.AddRenderRegion(num + 1 + num3, DiffResultSpanStatus.DeleteSource);
										}
										flag = true;
									}
								}
							}
							else if (!string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && !string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
							{
								if (!PreferenceManager.Current.Settings.SimplifyColor)
								{
									sourceViewer.AddRenderRegion(num + 1 + num3, DiffResultSpanStatus.Replace);
									targetViewer.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.Replace);
								}
								flag = true;
							}
							else if (!string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
							{
								if (!PreferenceManager.Current.Settings.SimplifyColor)
								{
									targetViewer.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.DeleteSource);
								}
								flag = true;
							}
							else if (string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && !string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
							{
								if (!PreferenceManager.Current.Settings.SimplifyColor)
								{
									sourceViewer.AddRenderRegion(num + 1 + num3, DiffResultSpanStatus.DeleteSource);
								}
								flag = true;
							}
						}
						if (withColor && flag)
						{
							list.Add(new DiffColor
							{
								LineNumber = num + 1 + num3
							});
						}
						num++;
						num2++;
					}
					break;
				}
				case DiffResultSpanStatus.DeleteSource:
				{
					bool flag = false;
					for (int k = 0; k < diffResultSpan.Length; k++)
					{
						if (num < diffList_TextFile.Count() - 1 || num2 < diffList_TextFile2.Count() - 1)
						{
							stringBuilder.Append(((TextLine)diffList_TextFile.GetByIndex(num)).Line + "\r\n");
							stringBuilder2.Append("\a\r\n");
						}
						else
						{
							stringBuilder.Append(((TextLine)diffList_TextFile.GetByIndex(num)).Line);
							stringBuilder2.Append("\a");
						}
						if (!string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")))
						{
							if (Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*#"))
							{
								if (!PreferenceManager.Current.Settings.SimplifyColor)
								{
									targetViewer.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.DeleteSource);
								}
								flag = true;
							}
							else
							{
								if (!PreferenceManager.Current.Settings.SimplifyColor)
								{
									targetViewer.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.DeleteSource);
								}
								flag = true;
							}
						}
						if (withColor && flag)
						{
							list.Add(new DiffColor
							{
								LineNumber = num + 1 + num3
							});
						}
						num4++;
						num++;
					}
					break;
				}
				case DiffResultSpanStatus.AddDestination:
				{
					bool flag = false;
					for (int l = 0; l < diffResultSpan.Length; l++)
					{
						if (num < diffList_TextFile.Count() - 1 || num2 < diffList_TextFile2.Count() - 1)
						{
							stringBuilder.Append("\a\r\n");
							stringBuilder2.Append(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line + "\r\n");
						}
						else
						{
							stringBuilder.Append("\a");
							stringBuilder2.Append(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line);
						}
						if (!string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
						{
							if (Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "^\\s*#"))
							{
								if (PreferenceManager.Current.Settings.DiffComment)
								{
									if (!PreferenceManager.Current.Settings.SimplifyColor)
									{
										sourceViewer.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.AddDestination);
									}
									flag = true;
								}
							}
							else
							{
								if (!PreferenceManager.Current.Settings.SimplifyColor)
								{
									sourceViewer.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.AddDestination);
								}
								flag = true;
							}
						}
						if (withColor && flag)
						{
							list.Add(new DiffColor
							{
								LineNumber = num + 1 + num3
							});
						}
						num3++;
						num2++;
					}
					break;
				}
				}
			}
			rep.Clear();
			sourceViewer.Text = stringBuilder.ToString();
			targetViewer.Text = stringBuilder2.ToString();
			if (this.MoveBetweenDiff)
			{
				ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).BaseOnStandardDiffColorArea = list;
			}
			sourceViewer.DiffCopy = list;
			if (PreferenceManager.Current.Settings.SimplifyColor)
			{
				HighlightingDiffColorize highlightingDiffColorize = new HighlightingDiffColorize();
				highlightingDiffColorize.DiffColorlist = list;
				targetViewer.TextArea.TextView.LineTransformers.Add(highlightingDiffColorize);
				sourceViewer.TextArea.TextView.LineTransformers.Add(highlightingDiffColorize);
			}
		}

		// Token: 0x0400009F RID: 159
		private IComparisonWindow View;

		// Token: 0x040000A0 RID: 160
		private DiffTextViewer _sourceEditor;

		// Token: 0x040000A1 RID: 161
		private DiffTextViewer _targeEditor;
	}
}
