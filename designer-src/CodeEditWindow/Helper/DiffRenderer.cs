using System;
using System.Collections.Generic;
using System.Windows;
using System.Windows.Media;
using DifferenceEngine;
using ICSharpCode.AvalonEdit;
using ICSharpCode.AvalonEdit.Document;
using ICSharpCode.AvalonEdit.Rendering;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x0200003F RID: 63
	public class DiffRenderer : IBackgroundRenderer
	{
		// Token: 0x17000065 RID: 101
		// (get) Token: 0x060002C5 RID: 709 RVA: 0x0001863F File Offset: 0x0001683F
		// (set) Token: 0x060002C6 RID: 710 RVA: 0x00018647 File Offset: 0x00016847
		public List<DiffResult> Results { get; set; }

		// Token: 0x17000066 RID: 102
		// (get) Token: 0x060002C7 RID: 711 RVA: 0x00018650 File Offset: 0x00016850
		public KnownLayer Layer
		{
			get
			{
				return KnownLayer.Background;
			}
		}

		// Token: 0x060002C8 RID: 712 RVA: 0x00018653 File Offset: 0x00016853
		public DiffRenderer(TextEditor editor)
		{
			this.Results = new List<DiffResult>();
			this._editor = editor;
		}

		// Token: 0x060002C9 RID: 713 RVA: 0x00018670 File Offset: 0x00016870
		private static Brush CreateYellowBackgroundBrush()
		{
			return new SolidColorBrush(Color.FromArgb(120, 238, byte.MaxValue, 0));
		}

		// Token: 0x060002CA RID: 714 RVA: 0x00018698 File Offset: 0x00016898
		private static Brush CreateGreenBackgroundBrush()
		{
			return new SolidColorBrush(Color.FromArgb(120, 0, byte.MaxValue, 0));
		}

		// Token: 0x060002CB RID: 715 RVA: 0x000186BC File Offset: 0x000168BC
		public void Draw(TextView textView, DrawingContext drawingContext)
		{
			try
			{
				foreach (DiffResult diffResult in this.Results)
				{
					if (diffResult.LineNumber > this._editor.Document.LineCount)
					{
						break;
					}
					DocumentLine lineByNumber = this._editor.Document.GetLineByNumber(diffResult.LineNumber);
					TextSegment textSegment = new TextSegment
					{
						StartOffset = lineByNumber.Offset,
						EndOffset = lineByNumber.EndOffset
					};
					switch (diffResult.Type)
					{
					case DiffResultSpanStatus.Replace:
					{
						using (IEnumerator<Rect> enumerator2 = BackgroundGeometryBuilder.GetRectsForSegment(textView, textSegment, false).GetEnumerator())
						{
							while (enumerator2.MoveNext())
							{
								Rect rect = enumerator2.Current;
								drawingContext.DrawRoundedRectangle(DiffRenderer._diffBrushYellow, null, new Rect(new Point(rect.Location.X + textView.HorizontalOffset, rect.Location.Y), new Size(textView.ActualWidth, rect.Height)), 1.0, 1.0);
							}
							continue;
						}
						break;
					}
					case DiffResultSpanStatus.DeleteSource:
						goto IL_01BD;
					case DiffResultSpanStatus.AddDestination:
						break;
					default:
						continue;
					}
					using (IEnumerator<Rect> enumerator3 = BackgroundGeometryBuilder.GetRectsForSegment(textView, textSegment, false).GetEnumerator())
					{
						while (enumerator3.MoveNext())
						{
							Rect rect2 = enumerator3.Current;
							drawingContext.DrawRoundedRectangle(DiffRenderer._diffBrushGreen, null, new Rect(new Point(rect2.Location.X + textView.HorizontalOffset, rect2.Location.Y), new Size(textView.ActualWidth, rect2.Height)), 1.0, 1.0);
						}
						continue;
					}
					IL_01BD:
					foreach (Rect rect3 in BackgroundGeometryBuilder.GetRectsForSegment(textView, textSegment, false))
					{
						drawingContext.DrawRoundedRectangle(DiffRenderer._diffBrushGreen, null, new Rect(new Point(rect3.Location.X + textView.HorizontalOffset, rect3.Location.Y), new Size(textView.ActualWidth, rect3.Height)), 1.0, 1.0);
					}
				}
			}
			catch
			{
			}
		}

		// Token: 0x060002CC RID: 716 RVA: 0x000189B8 File Offset: 0x00016BB8
		public void AddRenderRegion(int lineNumber, DiffResultSpanStatus type)
		{
			this.Results.Add(new DiffResult(lineNumber, type));
		}

		// Token: 0x060002CD RID: 717 RVA: 0x000189CC File Offset: 0x00016BCC
		public void Clear()
		{
			this.Results.Clear();
		}

		// Token: 0x04000137 RID: 311
		private TextEditor _editor;

		// Token: 0x04000138 RID: 312
		private static readonly Brush _diffBrushYellow = DiffRenderer.CreateYellowBackgroundBrush();

		// Token: 0x04000139 RID: 313
		private static readonly Brush _diffBrushGreen = DiffRenderer.CreateGreenBackgroundBrush();
	}
}
