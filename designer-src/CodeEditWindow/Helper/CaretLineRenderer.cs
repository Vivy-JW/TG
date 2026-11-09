using System;
using System.Windows;
using System.Windows.Media;
using ICSharpCode.AvalonEdit;
using ICSharpCode.AvalonEdit.Document;
using ICSharpCode.AvalonEdit.Rendering;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x0200001A RID: 26
	public class CaretLineRenderer : IBackgroundRenderer
	{
		// Token: 0x17000020 RID: 32
		// (get) Token: 0x06000104 RID: 260 RVA: 0x0000A7CA File Offset: 0x000089CA
		public KnownLayer Layer
		{
			get
			{
				return KnownLayer.Background;
			}
		}

		// Token: 0x06000105 RID: 261 RVA: 0x0000A7D0 File Offset: 0x000089D0
		private static Brush CreateBackgroundBrush()
		{
			return new SolidColorBrush(Color.FromArgb(65, 158, 158, 158));
		}

		// Token: 0x06000106 RID: 262 RVA: 0x0000A7FA File Offset: 0x000089FA
		public CaretLineRenderer(TextEditor editor)
		{
			this._editor = editor;
			this._editor.TextArea.Caret.PositionChanged += this.Caret_PositionChanged;
		}

		// Token: 0x06000107 RID: 263 RVA: 0x0000A82C File Offset: 0x00008A2C
		private void Caret_PositionChanged(object sender, EventArgs e)
		{
			DocumentLine lineByOffset = this._editor.Document.GetLineByOffset(this._editor.CaretOffset);
			if (lineByOffset.LineNumber != this.caretAtLine)
			{
				this._editor.TextArea.TextView.InvalidateLayer(this.Layer);
			}
			this.caretAtLine = lineByOffset.LineNumber;
		}

		// Token: 0x06000108 RID: 264 RVA: 0x0000A88C File Offset: 0x00008A8C
		public void Draw(TextView textView, DrawingContext drawingContext)
		{
			if (textView == null)
			{
				throw new ArgumentNullException("textView");
			}
			if (drawingContext == null)
			{
				throw new ArgumentNullException("drawingContext");
			}
			if (this._editor == null)
			{
				throw new ArgumentNullException("TextEditor");
			}
			textView.EnsureVisualLines();
			DocumentLine lineByOffset = this._editor.Document.GetLineByOffset(this._editor.CaretOffset);
			TextSegment textSegment = new TextSegment
			{
				StartOffset = lineByOffset.Offset,
				EndOffset = lineByOffset.EndOffset
			};
			foreach (Rect rect in BackgroundGeometryBuilder.GetRectsForSegment(textView, textSegment, false))
			{
				drawingContext.DrawRoundedRectangle(CaretLineRenderer._markerBrush, null, new Rect(new Point(rect.Location.X + textView.HorizontalOffset, rect.Location.Y), new Size(textView.ActualWidth, rect.Height)), 1.0, 1.0);
			}
		}

		// Token: 0x04000063 RID: 99
		private TextEditor _editor;

		// Token: 0x04000064 RID: 100
		private static readonly Brush _markerBrush = CaretLineRenderer.CreateBackgroundBrush();

		// Token: 0x04000065 RID: 101
		private int caretAtLine;
	}
}
