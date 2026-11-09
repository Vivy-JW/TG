using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Linq;
using System.Windows.Media;
using ICSharpCode.AvalonEdit.Document;
using ICSharpCode.AvalonEdit.Rendering;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000021 RID: 33
	public class TextMarkedRenderer : IBackgroundRenderer
	{
		// Token: 0x17000023 RID: 35
		// (get) Token: 0x06000136 RID: 310 RVA: 0x0000C051 File Offset: 0x0000A251
		public KnownLayer Layer
		{
			get
			{
				return KnownLayer.Selection;
			}
		}

		// Token: 0x06000137 RID: 311 RVA: 0x0000C054 File Offset: 0x0000A254
		private static Brush CreateBackgroundBrush()
		{
			return new SolidColorBrush(Color.FromArgb(byte.MaxValue, 200, 200, 200));
		}

		// Token: 0x06000139 RID: 313 RVA: 0x0000C094 File Offset: 0x0000A294
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
			if (!textView.VisualLinesValid)
			{
				return;
			}
			ReadOnlyCollection<VisualLine> visualLines = textView.VisualLines;
			if (visualLines.Count == 0)
			{
				return;
			}
			int offset = visualLines.First<VisualLine>().FirstDocumentLine.Offset;
			int endOffset = visualLines.Last<VisualLine>().LastDocumentLine.EndOffset;
			BackgroundGeometryBuilder backgroundGeometryBuilder = new BackgroundGeometryBuilder();
			backgroundGeometryBuilder.AlignToMiddleOfPixels = true;
			backgroundGeometryBuilder.CornerRadius = 1.0;
			foreach (MarkedSegment markedSegment in this._list)
			{
				DocumentLine documentLine = markedSegment.StartLine;
				while (documentLine != null && documentLine.EndOffset <= endOffset && documentLine.EndOffset <= markedSegment.EndLine.EndOffset)
				{
					backgroundGeometryBuilder.AddSegment(textView, new TextSegment
					{
						StartOffset = documentLine.Offset,
						EndOffset = documentLine.EndOffset
					});
					documentLine = documentLine.NextLine;
				}
			}
			Geometry geometry = backgroundGeometryBuilder.CreateGeometry();
			if (geometry != null)
			{
				drawingContext.DrawGeometry(TextMarkedRenderer._markerBrush, null, geometry);
			}
		}

		// Token: 0x0600013A RID: 314 RVA: 0x0000C1D0 File Offset: 0x0000A3D0
		public void AppendMarkedSegment(MarkedSegment markedSegment)
		{
			this._list.Add(markedSegment);
		}

		// Token: 0x0600013B RID: 315 RVA: 0x0000C1DE File Offset: 0x0000A3DE
		public void ClearMarkedSegment()
		{
			this._list.Clear();
		}

		// Token: 0x0600013C RID: 316 RVA: 0x0000C1EC File Offset: 0x0000A3EC
		public void RemoveMarkedSegment(MarkedSegment markedSegment)
		{
			MarkedSegmentComparer markedSegmentComparer = new MarkedSegmentComparer();
			for (int i = 0; i < this._list.Count; i++)
			{
				MarkedSegment markedSegment2 = this._list[i];
				if (markedSegmentComparer.Compare(markedSegment, markedSegment2) == 0)
				{
					this._list.Remove(markedSegment2);
				}
			}
		}

		// Token: 0x04000099 RID: 153
		private List<MarkedSegment> _list = new List<MarkedSegment>();

		// Token: 0x0400009A RID: 154
		private static readonly Brush _markerBrush = TextMarkedRenderer.CreateBackgroundBrush();
	}
}
