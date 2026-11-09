using System;
using ICSharpCode.AvalonEdit.Document;
using SpecDesigner.CodeEditWindow.View;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x0200003E RID: 62
	public static class CodeEditorExtension
	{
		// Token: 0x060002C4 RID: 708 RVA: 0x000185C0 File Offset: 0x000167C0
		public static MarkedSegment FindMarkSegment(this CodeTextEditor editor, int offset)
		{
			DocumentLine lineByOffset = editor.Document.GetLineByOffset(offset);
			SegmentObject segmentObject = editor.Document.SectionProvider.Find(lineByOffset.LineNumber);
			SegmentObject nextNode = segmentObject.NextNode;
			DocumentLine lineByNumber = editor.Document.GetLineByNumber(segmentObject.EndOffset + 1);
			DocumentLine lineByNumber2 = editor.Document.GetLineByNumber(nextNode.EndOffset - 1);
			return new MarkedSegment
			{
				StartLine = lineByNumber,
				EndLine = lineByNumber2
			};
		}
	}
}
